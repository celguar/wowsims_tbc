package feral

import (
	"time"

	"github.com/wowsims/tbc/sim/core"
	"github.com/wowsims/tbc/sim/core/items"
	"github.com/wowsims/tbc/sim/core/proto"
	"github.com/wowsims/tbc/sim/druid"
)

func (cat *FeralDruid) OnGCDReady(sim *core.Simulation) {
	cat.doRotation(sim)
}

// Ported from https://github.com/NerdEgghead/TBC_cat_sim

func (cat *FeralDruid) shift(sim *core.Simulation) bool {
	cat.waitingForTick = false

	// If we have just now decided to shift, then we do not execute the
	// shift immediately, but instead trigger an input delay for realism.
	if !cat.readyToShift {
		cat.readyToShift = true
		return false
	}

	cat.readyToShift = false
	return cat.PowerShiftCat(sim)
}

func (cat *FeralDruid) doRotation(sim *core.Simulation) bool {
	// On gcd do nothing
	if !cat.GCD.IsReady(sim) {
		return false
	}

	// If we're out of form always shift back in
	if !cat.InForm(druid.Cat) {
		return cat.CatForm.Cast(sim, nil)
	}

	// If we previously decided to shift, then execute the shift now once
	// the input delay is over.
	if cat.readyToShift {
		return cat.shift(sim)
	}

	rotation := &cat.Rotation

	energy := cat.CurrentEnergy()
	cp := cat.ComboPoints()
	rip_debuff := cat.RipDot.IsActive()
	rip_end := cat.RipDot.ExpiresAt()
	mangle_debuff := cat.MangleAura.IsActive()
	mangle_end := cat.MangleAura.ExpiresAt()
	rake_debuff := cat.RakeDot.IsActive()
	next_tick := cat.NextEnergyTickAt()
	shift_cost := cat.CatForm.DefaultCast.Cost
	omen_proc := cat.ClearcastingActive()

	// 10/6/21 - Added logic to not cast Rip if we're near the end of the
	// fight.

	rip_now := cp >= rotation.rip_cp && !rip_debuff
	ripweave_now := (rotation.use_rip_trick &&
		cp >= rotation.rip_trick_cp &&
		!rip_debuff &&
		energy >= rip_trick_min &&
		!cat.PseudoStats.NoCost)

	rip_now = (rip_now || ripweave_now) && (sim.Duration-sim.CurrentTime >= rip_end_thresh)

	bite_at_end := (cp >= rotation.bite_cp &&
		((sim.Duration-sim.CurrentTime < rip_end_thresh) ||
			(rip_debuff && (sim.Duration-rip_end < rip_end_thresh))))

	mangle_now := !rip_now && !mangle_debuff
	mangle_cost := cat.Mangle.DefaultCast.Cost

	bite_before_rip := (rip_debuff && rotation.use_bite &&
		(rip_end-sim.CurrentTime >= bite_time))

	bite_now := ((bite_before_rip || rotation.bite_over_rip) &&
		cp >= rotation.bite_cp)

	rip_next := ((rip_now || ((cp >= rotation.rip_cp) && (rip_end <= next_tick))) &&
		(sim.Duration-next_tick >= rip_end_thresh))

	mangle_next := (!rip_next && (mangle_now || mangle_end <= next_tick))

	// 12/2/21 - Added wait_to_mangle parameter that tells us whether we
	// should wait for the next Energy tick and cast Mangle, assuming we
	// are less than a tick's worth of Energy from being able to cast it. In
	// a standard Wolfshead rotation, wait_for_mangle is identical to
	// mangle_next, i.e. we only wait for the tick if Mangle will have
	// fallen off before the next tick. In a no-Wolfshead rotation, however,
	// it is preferable to Mangle rather than Shred as the second special in
	// a standard cycle, provided a bonus like 2pT6 is present to bring the
	// Mangle Energy cost down to 38 or below so that it can be fit in
	// alongside a Shred.
	wait_to_mangle := (mangle_next || ((!rotation.wolfshead) && (mangle_cost <= 38)))

	bite_before_rip_next := (bite_before_rip &&
		(rip_end-next_tick >= bite_time))

	prio_bite_over_mangle := (rotation.bite_over_rip || (!mangle_now))

	time_to_next_tick := next_tick - sim.CurrentTime
	cat.waitingForTick = true

	if cat.CurrentMana() < shift_cost {
		// If this is the first time we're oom, log it
		//if self.time_to_oom is None:
		//    self.time_to_oom = time //TODO

		// No-shift rotation
		if rip_now && ((energy >= 30) || omen_proc) {
			cat.Rip.Cast(sim, cat.CurrentTarget)
			cat.waitingForTick = false
		} else if mangle_now &&
			((energy >= mangle_cost) || omen_proc) {
			return cat.Mangle.Cast(sim, cat.CurrentTarget)
		} else if bite_now && ((energy >= 35) || omen_proc) {
			return cat.FerociousBite.Cast(sim, cat.CurrentTarget)
		} else if (energy >= 42) || omen_proc {
			return cat.Shred.Cast(sim, cat.CurrentTarget)
		}
	} else if energy < 10 {
		cat.shift(sim)
	} else if rip_now {
		if (energy >= 30) || omen_proc {
			cat.Rip.Cast(sim, cat.CurrentTarget)
			cat.waitingForTick = false
		} else if time_to_next_tick > max_wait_time {
			cat.shift(sim)
		}
	} else if (bite_now || bite_at_end) && prio_bite_over_mangle {
		// Decision tree for Bite usage is more complicated, so there is
		// some duplicated logic with the main tree.

		// Shred versus Bite decision is the same as vanilla criteria.

		// Bite immediately if we'd have to wait for the following cast.
		cutoff_mod := 20.0
		if time_to_next_tick <= time.Second {
			cutoff_mod = 0.0
		}
		if (energy >= 57.0+cutoff_mod) ||
			((energy >= 15+cutoff_mod) && omen_proc) {
			return cat.Shred.Cast(sim, cat.CurrentTarget)
		}
		if energy >= 35 {
			return cat.FerociousBite.Cast(sim, cat.CurrentTarget)
		}
		// If we are doing the Rip rotation with Bite filler, then there is
		// a case where we would Bite now if we had enough energy, but once
		// we gain enough energy to do so, it's too late to Bite relative to
		// Rip falling off. In this case, we wait for the tick only if we
		// can Shred or Mangle afterward, and otherwise shift and won't Bite
		// at all this cycle. Returning 0.0 is the same thing as waiting for
		// the next tick, so this logic could be written differently if
		// desired to match the rest of the rotation code, where waiting for
		// tick is handled implicitly instead.
		wait := false
		if (energy >= 22) && bite_before_rip &&
			(!bite_before_rip_next) {
			wait = true
		} else if (energy >= 15) &&
			((!bite_before_rip) ||
				bite_before_rip_next || bite_at_end) {
			wait = true
		} else if (!rip_next) && ((energy < 20) || (!mangle_next)) {
			wait = false
			cat.shift(sim)
		} else {
			wait = true
		}
		if wait && (time_to_next_tick > max_wait_time) {
			cat.shift(sim)
		}
	} else if energy >= 35 && energy <= bite_trick_max &&
		rotation.use_rake_trick &&
		(time_to_next_tick > cat.latency) &&
		!omen_proc &&
		cp >= bite_trick_cp {
		return cat.FerociousBite.Cast(sim, cat.CurrentTarget)
	} else if energy >= 35 && energy < mangle_cost &&
		rotation.use_rake_trick &&
		(time_to_next_tick > 1*time.Second+cat.latency) &&
		!rake_debuff &&
		!omen_proc {
		return cat.Rake.Cast(sim, cat.CurrentTarget)
	} else if mangle_now {
		if (energy < mangle_cost-20) && (!rip_next) {
			cat.shift(sim)
		} else if (energy >= mangle_cost) || omen_proc {
			return cat.Mangle.Cast(sim, cat.CurrentTarget)
		} else if time_to_next_tick > max_wait_time {
			cat.shift(sim)
		}
	} else if energy >= 22 {
		if omen_proc {
			return cat.Shred.Cast(sim, cat.CurrentTarget)
		}
		// If our energy value is between 50-56 with 2pT6, or 60-61 without,
		// and we are within 1 second of an Energy tick, then Shredding now
		// forces us to shift afterwards, whereas we can instead cast two
		// Mangles instead for higher cpm. This scenario is most relevant
		// when using a no-Wolfshead rotation with 2pT6, and it will
		// occur whenever the initial Shred on a cycle misses.
		if (energy >= 2*mangle_cost-20) && (energy < 22+mangle_cost) &&
			(time_to_next_tick <= 1.0*time.Second) &&
			rotation.use_mangle_trick &&
			(!rotation.use_rake_trick || mangle_cost == 35) {
			return cat.Mangle.Cast(sim, cat.CurrentTarget)
		}
		if energy >= 42 {
			return cat.Shred.Cast(sim, cat.CurrentTarget)
		}
		if (energy >= mangle_cost) &&
			(time_to_next_tick > time.Second+cat.latency) {
			return cat.Mangle.Cast(sim, cat.CurrentTarget)
		}
		if time_to_next_tick > max_wait_time {
			cat.shift(sim)
		}
	} else if (!rip_next) && ((energy < mangle_cost-20) || (!wait_to_mangle)) {
		cat.shift(sim)
	} else if time_to_next_tick > max_wait_time {
		cat.shift(sim)
	}
	// Model two types of input latency: (1) When waiting for an energy tick
	// to execute the next special ability, the special will in practice be
	// slightly delayed after the tick arrives. (2) When executing a
	// powershift without clipping the GCD, the shift will in practice be
	// slightly delayed after the GCD ends.

	if cat.readyToShift {
		cat.SetGCDTimer(sim, sim.CurrentTime+cat.latency)
	} else if cat.waitingForTick {
		cat.SetGCDTimer(sim, sim.CurrentTime+time_to_next_tick+cat.latency)
	}

	return false
}

const bite_trick_cp = int32(2)
const bite_trick_max = 39.0
const bite_time = time.Second * 0.0
const rip_trick_min = 52.0
const rip_end_thresh = time.Second * 10
const max_wait_time = time.Second * 1.0

type FeralDruidRotation struct {
	rip_cp           int32
	bite_cp          int32
	rip_trick_cp     int32
	use_bite         bool
	bite_over_rip    bool
	use_mangle_trick bool
	use_rip_trick    bool
	use_rake_trick   bool
	wolfshead        bool
}

func (cat *FeralDruid) setupRotation(rotation *proto.FeralDruid_Rotation) {

	use_bite := (rotation.Biteweave && rotation.FinishingMove == proto.FeralDruid_Rotation_Rip) ||
		rotation.FinishingMove == proto.FeralDruid_Rotation_Bite
	rip_cp := rotation.RipMinComboPoints

	if rotation.FinishingMove != proto.FeralDruid_Rotation_Rip {
		rip_cp = 6
	}

	cat.Rotation = FeralDruidRotation{
		rip_cp:           rip_cp,
		bite_cp:          rotation.BiteMinComboPoints,
		rip_trick_cp:     rotation.RipMinComboPoints,
		use_bite:         use_bite,
		bite_over_rip:    use_bite && rotation.FinishingMove != proto.FeralDruid_Rotation_Rip,
		use_mangle_trick: rotation.MangleTrick,
		use_rip_trick:    rotation.Ripweave,
		use_rake_trick:   rotation.RakeTrick && !druid.ItemSetThunderheartHarness.CharacterHasSetBonus(&cat.Character, 2),
		wolfshead:        cat.Equip[items.ItemSlotHead].ID == 8345,
	}

}
