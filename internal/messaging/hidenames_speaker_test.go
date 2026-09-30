package messaging

import "testing"

// Both hiders fit the one type the room senders take.
var (
	_ NameHider = HideNames
	_ NameHider = HideSpeakerNames
)

const speakerLine = `<ansi fg="username">Kesh</ansi> says, "<ansi fg="saytext">I am Kesh</ansi>"`

func TestHideSpeakerNames_ClearSightKeepsTheName(t *testing.T) {
	if got := HideSpeakerNames(speakerLine, []string{"Kesh"}, SightFull); got != speakerLine {
		t.Fatalf("clear sight changed the line: %q", got)
	}
}

func TestHideSpeakerNames_ShapesHearAFigure(t *testing.T) {
	want := `<ansi fg="combat-anon">A figure</ansi> says, "<ansi fg="saytext">I am Kesh</ansi>"`
	if got := HideSpeakerNames(speakerLine, []string{"Kesh"}, SightShapes); got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

// Ruling 3: an unseen speaker is "Someone", and the words arrive untouched,
// even when they say the speaker's own name.
func TestHideSpeakerNames_NoSightHearsSomeoneAndEveryWord(t *testing.T) {
	want := `<ansi fg="combat-anon">Someone</ansi> says, "<ansi fg="saytext">I am Kesh</ansi>"`
	if got := HideSpeakerNames(speakerLine, []string{"Kesh"}, SightNone); got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestHideSpeakerNames_MobTagWithDuplicateIndex(t *testing.T) {
	line := `<ansi fg="mobname">guard #2</ansi> shouts, "<ansi fg="saytext-mob">HALT</ansi>"`
	want := `<ansi fg="combat-anon">Someone</ansi> shouts, "<ansi fg="saytext-mob">HALT</ansi>"`
	if got := HideSpeakerNames(line, []string{"guard"}, SightNone); got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestHideSpeakerNames_NoNameIsIgnored(t *testing.T) {
	if got := HideSpeakerNames(speakerLine, []string{NoName}, SightNone); got != speakerLine {
		t.Fatalf("NoName hid something: %q", got)
	}
}
