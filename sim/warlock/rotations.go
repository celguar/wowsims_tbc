package warlock

import (
	"time"

	"github.com/wowsims/tbc/sim/core"
	"github.com/wowsims/tbc/sim/core/proto"
	"github.com/wowsims/tbc/sim/core/stats"
)

func (warlock *Warlock) OnGCDReady(sim *core.Simulation) {
	warlock.tryUseGCD(sim)
}

func (warlock *Warlock) OnManaTick(sim *core.Simulation) {
	if warlock.FinishedWaitingForManaAndGCDReady(sim) {
		warlock.tryUseGCD(sim)
	}
}

func (warlock *Warlock) tryUseGCD(sim *core.Simulation) {

	var spell *core.Spell
	var target = sim.GetPrimaryTarget()

	// Apply curses first
	// TODO: should this be part of setup instead of during main rotation?
	castCurse := func(spellToCast *core.Spell, aura *core.Aura) bool {
		if !aura.IsActive() {
			spell = spellToCast
			return true
		}
		return false
	}
	switch warlock.Rotation.Curse {
	case proto.Warlock_Rotation_Elements:
		castCurse(warlock.CurseOfElements, warlock.CurseOfElementsAura)
	case proto.Warlock_Rotation_Recklessness:
		castCurse(warlock.CurseOfRecklessness, warlock.CurseOfRecklessnessAura)
	case proto.Warlock_Rotation_Tongues:
		castCurse(warlock.CurseOfTongues, warlock.CurseOfTonguesAura)
	case proto.Warlock_Rotation_Doom:
		if warlock.Talents.AmplifyCurse && warlock.AmplifyCurse.CD.IsReady(sim) {
			warlock.AmplifyCurse.Cast(sim, sim.GetPrimaryTarget())
		}
		if sim.Duration-sim.CurrentTime < time.Minute {
			if !warlock.CurseOfAgonyDot.IsActive() {
				spell = warlock.CurseOfAgony
			}
		} else if warlock.CurseOfDoom.CD.IsReady(sim) && !warlock.CurseOfDoomDot.IsActive() {
			spell = warlock.CurseOfDoom
		}
	case proto.Warlock_Rotation_Agony:
		if !warlock.CurseOfAgonyDot.IsActive() {
			spell = warlock.CurseOfAgony
		}
	}
	if spell != nil {
		if !spell.Cast(sim, target) {
			warlock.LifeTap.Cast(sim, target)
		}
		return
	}

	bigCDs := warlock.GetMajorCooldowns()
	nextBigCD := sim.Duration
	for _, cd := range bigCDs {
		if cd == nil {
			continue // not on cooldown right now.
		}
		cdReadyAt := cd.Cooldown.ReadyAt()
		if cd.Type == core.CooldownTypeDPS && cdReadyAt < nextBigCD {
			nextBigCD = cdReadyAt
		}
	}

	// If big CD coming up and we don't have enough mana for it, lifetap
	// Also, never do a big regen in the last few seconds of the fight.
	if !warlock.DoingRegen && nextBigCD-sim.CurrentTime < time.Second*15 && sim.Duration-sim.CurrentTime > time.Second*20 {
		if warlock.GetStat(stats.SpellPower) > warlock.GetInitialStat(stats.SpellPower) || warlock.CastSpeed() > warlock.InitialCastSpeed() {
			// never start regen if you have boosted sp or boosted cast speed
		} else if warlock.CurrentManaPercent() < 0.2 {
			warlock.DoingRegen = true
		}
	}

	if warlock.DoingRegen {
		if nextBigCD-sim.CurrentTime < time.Second*2 {
			// stop regen, start blasting
			warlock.DoingRegen = false
		} else {
			warlock.LifeTap.Cast(sim, target)
			if warlock.CurrentManaPercent() > 0.6 {
				warlock.DoingRegen = false
			}
			return
		}
	}

	// main spells
	// TODO: optimize so that cast time of DoT is included in calculation so you can cast right before falling off.
	if warlock.Talents.UnstableAffliction && !warlock.UnstableAffDot.IsActive() {
		spell = warlock.UnstableAff
	} else if warlock.Rotation.Corruption && !warlock.CorruptionDot.IsActive() {
		spell = warlock.Corruption
	} else if warlock.Talents.SiphonLife && !warlock.SiphonLifeDot.IsActive() && warlock.ImpShadowboltAura.IsActive() {
		spell = warlock.SiphonLife
	} else if warlock.Rotation.Immolate && !warlock.ImmolateDot.IsActive() {
		spell = warlock.Immolate
	} else {
		switch warlock.Rotation.PrimarySpell {
		case proto.Warlock_Rotation_Shadowbolt:
			spell = warlock.Shadowbolt
		case proto.Warlock_Rotation_Incinerate:
			spell = warlock.Incinerate
		default:
			panic("no primary spell set")
		}
	}

	if success := spell.Cast(sim, target); success {
		return
	}

	// If we were not successful at anything else, lifetap.
	warlock.LifeTap.Cast(sim, target)
}