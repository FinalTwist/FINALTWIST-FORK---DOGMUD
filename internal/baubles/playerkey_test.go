package baubles

import (
	"errors"
	"testing"
)

// Text a player's own key wrote reaches other players, so it is held to
// ASCII letters, spaces and ' " - , . ! ? with every run of periods followed
// by a space, a " or the end (spec S3, ruling 15). Cyrillic lookalikes pass
// CleanReply (they are real letters) and are refused here. The check takes
// CLEANED text: an em dash is refused here and folded to - by CleanReply
// first (TestPlayerKeyTextIsFoldedBeforeTheCheck).
func TestCheckPlayerKeyText(t *testing.T) {
	if err := CheckPlayerKeyText(goodReply()); err != nil {
		t.Fatalf("the good reply is plain text: %v", err)
	}
	cases := map[string]func(*Reply){
		`cyrillic a in the name`: func(r *Reply) { r.Name = "P\U00000430inted Wooden Horse" },
		`accent`:                 func(r *Reply) { r.Description = "A caf\U000000E9 toy horse, its red paint flaking from the mane." },
		`digit`:                  func(r *Reply) { r.Description = `A toy horse, 3 legs left, its red paint flaking away.` },
		`colon`:                  func(r *Reply) { r.Description = `A toy horse: its red paint is flaking from the mane.` },
		`semicolon`:              func(r *Reply) { r.Description = `A toy horse; its red paint is flaking from the mane.` },
		`uncleaned em dash`:      func(r *Reply) { r.Description = "A toy horse \U00002014 its red paint flaking from the mane." },
		`inner period`:           func(r *Reply) { r.Description = `A toy horse.Its red paint is flaking from the mane.` },
		`ellipsis glued on`:      func(r *Reply) { r.Description = `A toy horse...its red paint flaking from the mane.` },
		`material`:               func(r *Reply) { r.Material = `pine/oak` },
		`name simple`:            func(r *Reply) { r.NameSimple = "h\U000000F6rse" },
	}
	for name, change := range cases {
		r := goodReply()
		change(&r)
		if err := CheckPlayerKeyText(r); !errors.Is(err, ErrUnusableReply) {
			t.Errorf("%s must be refused, got %v", name, err)
		}
	}
	r := goodReply()
	r.Description = `Old, chipped - and loved! Whose was it? Nobody's now... "Mine." it says.`
	if err := CheckPlayerKeyText(r); err != nil {
		t.Fatalf("every allowed mark together passes: %v", err)
	}
	// A homoglyph name passes CleanReply: the defence is this check.
	r = goodReply()
	r.Name = "P\U00000430inted Wooden Horse"
	if _, err := CleanReply(r); err != nil {
		t.Fatalf("a Cyrillic letter is a letter to CleanReply: %v", err)
	}
}

// Curly quotes, an em dash, an en dash and the ellipsis pass once
// CleanReply has folded them (owner ruling 15): a model's ordinary
// typography does not cost a player-key find its name.
func TestPlayerKeyTextIsFoldedBeforeTheCheck(t *testing.T) {
	r := goodReply()
	r.Name = "Mara\U00002019s Wooden Horse"
	r.Description = "A child\U00002018s toy horse \U00002014 its paint flaking \U00002013 marked \U0000201CMara\U0000201D\U00002026 still loved."
	cleaned, err := CleanReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckPlayerKeyText(cleaned); err != nil {
		t.Fatalf("folded typography passes the allowlist: %v", err)
	}
}
