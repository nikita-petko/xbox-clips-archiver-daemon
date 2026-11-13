package dl

import (
	"testing"

	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/users"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/xbl/models"
)

func TestDownloadMediaPreservesGamertag(t *testing.T) {
	user := users.User{UserID: "123", Gamertag: "Player"}
	items, err := downloadAllMediaForUser(user, []*models.Media{{Id: "clip"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Gamertag != user.Gamertag {
		t.Fatalf("downloaded media = %#v", items)
	}
}
