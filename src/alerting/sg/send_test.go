package sg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func TestConcurrentSendMail(t *testing.T) {
	const count = 30
	subjects := make(chan string, count)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var message struct {
			Subject string `json:"subject"`
		}
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			t.Errorf("decode message: %v", err)
		} else {
			subjects <- message.Subject
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	oldClient, oldSender, oldMailingList := gSendGridClient, gSenderEmail, gMailingList
	t.Cleanup(func() {
		gSendGridClient, gSenderEmail, gMailingList = oldClient, oldSender, oldMailingList
	})
	gSendGridClient = sendgrid.NewSendClient("test")
	gSendGridClient.BaseURL = server.URL
	gSenderEmail = mail.NewEmail("Sender", "sender@example.com")
	gMailingList = []*mail.Email{mail.NewEmail("Receiver", "receiver@example.com")}

	var wg sync.WaitGroup
	for i := range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			SendMail(context.Background(), fmt.Sprintf("user-%d", i), "body")
		}()
	}
	wg.Wait()
	close(subjects)
	got := make(map[string]int)
	for subject := range subjects {
		got[subject]++
	}
	for i := range count {
		subject := fmt.Sprintf("user-%d", i)
		if got[subject] != 1 {
			t.Errorf("subject %q received %d times", subject, got[subject])
		}
	}
}
