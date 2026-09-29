package users

import "testing"

// A link-dead player's character is still in the world, so the bauble
// sweep must see it: GetAllLoadedUsers includes zombies.
func TestGetAllLoadedUsersIncludesZombies(t *testing.T) {
	userManager.mu.Lock()
	userManager.Users[990001] = &UserRecord{UserId: 990001}
	userManager.Users[990002] = &UserRecord{UserId: 990002, isZombie: true}
	userManager.mu.Unlock()
	t.Cleanup(func() {
		userManager.mu.Lock()
		delete(userManager.Users, 990001)
		delete(userManager.Users, 990002)
		userManager.mu.Unlock()
	})

	found := map[int]bool{}
	for _, u := range GetAllLoadedUsers() {
		found[u.UserId] = true
	}
	if !found[990001] || !found[990002] {
		t.Fatalf("loaded users %v, want both the active and the zombie", found)
	}
	for _, u := range GetAllActiveUsers() {
		if u.UserId == 990002 {
			t.Fatal("GetAllActiveUsers still skips zombies")
		}
	}
}
