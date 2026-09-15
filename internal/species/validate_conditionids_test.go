package species

import "testing"

// The guard that would have caught condition 29 on the day it broke: a species
// referencing a condition id with no definition must fail the boot, not run for
// months with 67 mobs silently blind.
func TestValidateSpeciesConditionIdsPanicsOnMissingCondition(t *testing.T) {
	orig := allSpecies
	t.Cleanup(func() { allSpecies = orig })
	allSpecies = map[int]*Species{
		99: {SpeciesId: 99, Name: "Fixture", ConditionIds: []int{123456}},
	}

	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for a species referencing a condition that does not exist")
		}
	}()
	ValidateSpeciesConditionIds(func(int) bool { return false })
}

func TestValidateSpeciesConditionIdsAcceptsKnownConditions(t *testing.T) {
	orig := allSpecies
	t.Cleanup(func() { allSpecies = orig })
	allSpecies = map[int]*Species{
		99: {SpeciesId: 99, Name: "Fixture", ConditionIds: []int{29}},
	}
	ValidateSpeciesConditionIds(func(id int) bool { return id == 29 })
}

// A species with no buffids at all must not trip the guard.
func TestValidateSpeciesConditionIdsIgnoresSpeciesWithNoConditions(t *testing.T) {
	orig := allSpecies
	t.Cleanup(func() { allSpecies = orig })
	allSpecies = map[int]*Species{
		98: {SpeciesId: 98, Name: "Plain"},
	}
	ValidateSpeciesConditionIds(func(int) bool { return false })
}
