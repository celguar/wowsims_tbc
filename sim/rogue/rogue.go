package rogue

import (
	"github.com/wowsims/tbc/sim/core"
	"github.com/wowsims/tbc/sim/core/proto"
	"github.com/wowsims/tbc/sim/core/stats"
)

func RegisterRogue() {
	core.RegisterAgentFactory(
		proto.Player_Rogue{},
		proto.Spec_SpecRogue,
		func(character core.Character, options proto.Player) core.Agent {
			return NewRogue(character, options)
		},
		func(player *proto.Player, spec interface{}) {
			playerSpec, ok := spec.(*proto.Player_Rogue)
			if !ok {
				panic("Invalid spec value for Rogue!")
			}
			player.Spec = playerSpec
		},
	)
}

type Rogue struct {
	core.Character

	Talents  proto.RogueTalents
	Options  proto.Rogue_Options
	Rotation proto.Rogue_Rotation
}

func (rogue *Rogue) GetCharacter() *core.Character {
	return &rogue.Character
}

func (rogue *Rogue) GetRogue() *Rogue {
	return rogue
}

func (rogue *Rogue) AddRaidBuffs(raidBuffs *proto.RaidBuffs)    {}
func (rogue *Rogue) AddPartyBuffs(partyBuffs *proto.PartyBuffs) {}

func (rogue *Rogue) Init(sim *core.Simulation) {
}

func (rogue *Rogue) Reset(sim *core.Simulation) {
}

func NewRogue(character core.Character, options proto.Player) *Rogue {
	rogueOptions := options.GetRogue()

	rogue := &Rogue{
		Character: character,
		Talents:   *rogueOptions.Talents,
		Options:   *rogueOptions.Options,
		Rotation:  *rogueOptions.Rotation,
	}

	rogue.EnableEnergyBar(100, func(sim *core.Simulation) {
	})
	rogue.EnableAutoAttacks(rogue, core.AutoAttackOptions{
		// TODO: Crit multiplier
		MainHand:       rogue.WeaponFromMainHand(rogue.DefaultMeleeCritMultiplier()),
		OffHand:        rogue.WeaponFromOffHand(rogue.DefaultMeleeCritMultiplier()),
		AutoSwingMelee: true,
	})

	rogue.AddStatDependency(stats.StatDependency{
		SourceStat:   stats.Strength,
		ModifiedStat: stats.AttackPower,
		Modifier: func(strength float64, attackPower float64) float64 {
			return attackPower + strength*1
		},
	})

	rogue.AddStatDependency(stats.StatDependency{
		SourceStat:   stats.Agility,
		ModifiedStat: stats.AttackPower,
		Modifier: func(agility float64, attackPower float64) float64 {
			return attackPower + agility*1
		},
	})

	rogue.AddStatDependency(stats.StatDependency{
		SourceStat:   stats.Agility,
		ModifiedStat: stats.MeleeCrit,
		Modifier: func(agility float64, meleeCrit float64) float64 {
			return meleeCrit + (agility/40)*core.MeleeCritRatingPerCritChance
		},
	})

	//rogue.applyTalents()

	return rogue
}

func init() {
	core.BaseStats[core.BaseStatsKey{Race: proto.Race_RaceBloodElf, Class: proto.Class_ClassRogue}] = stats.Stats{
		stats.Strength:  92,
		stats.Agility:   160,
		stats.Stamina:   88,
		stats.Intellect: 43,
		stats.Spirit:    57,

		stats.AttackPower: 120,
		stats.MeleeCrit:   -0.3 * core.MeleeCritRatingPerCritChance,
	}
	core.BaseStats[core.BaseStatsKey{Race: proto.Race_RaceDwarf, Class: proto.Class_ClassRogue}] = stats.Stats{
		stats.Strength:  97,
		stats.Agility:   154,
		stats.Stamina:   92,
		stats.Intellect: 38,
		stats.Spirit:    57,

		stats.AttackPower: 120,
		stats.MeleeCrit:   -0.3 * core.MeleeCritRatingPerCritChance,
	}
	core.BaseStats[core.BaseStatsKey{Race: proto.Race_RaceGnome, Class: proto.Class_ClassRogue}] = stats.Stats{
		stats.Strength:  90,
		stats.Agility:   161,
		stats.Stamina:   88,
		stats.Intellect: 45,
		stats.Spirit:    58,

		stats.AttackPower: 120,
		stats.MeleeCrit:   -0.3 * core.MeleeCritRatingPerCritChance,
	}
	core.BaseStats[core.BaseStatsKey{Race: proto.Race_RaceHuman, Class: proto.Class_ClassRogue}] = stats.Stats{
		stats.Strength:  95,
		stats.Agility:   158,
		stats.Stamina:   89,
		stats.Intellect: 39,
		stats.Spirit:    58,

		stats.AttackPower: 120,
		stats.MeleeCrit:   -0.3 * core.MeleeCritRatingPerCritChance,
	}
	core.BaseStats[core.BaseStatsKey{Race: proto.Race_RaceNightElf, Class: proto.Class_ClassRogue}] = stats.Stats{
		stats.Strength:  92,
		stats.Agility:   163,
		stats.Stamina:   88,
		stats.Intellect: 39,
		stats.Spirit:    58,

		stats.AttackPower: 120,
		stats.MeleeCrit:   -0.3 * core.MeleeCritRatingPerCritChance,
	}
	core.BaseStats[core.BaseStatsKey{Race: proto.Race_RaceOrc, Class: proto.Class_ClassRogue}] = stats.Stats{
		stats.Strength:  98,
		stats.Agility:   155,
		stats.Stamina:   91,
		stats.Intellect: 36,
		stats.Spirit:    61,

		stats.AttackPower: 120,
		stats.MeleeCrit:   -0.3 * core.MeleeCritRatingPerCritChance,
	}
	trollStats := stats.Stats{
		stats.Strength:  96,
		stats.Agility:   160,
		stats.Stamina:   90,
		stats.Intellect: 35,
		stats.Spirit:    59,

		stats.AttackPower: 120,
		stats.MeleeCrit:   -0.3 * core.MeleeCritRatingPerCritChance,
	}
	core.BaseStats[core.BaseStatsKey{Race: proto.Race_RaceTroll10, Class: proto.Class_ClassRogue}] = trollStats
	core.BaseStats[core.BaseStatsKey{Race: proto.Race_RaceTroll30, Class: proto.Class_ClassRogue}] = trollStats
	core.BaseStats[core.BaseStatsKey{Race: proto.Race_RaceUndead, Class: proto.Class_ClassRogue}] = stats.Stats{
		stats.Strength:  94,
		stats.Agility:   156,
		stats.Stamina:   90,
		stats.Intellect: 37,
		stats.Spirit:    63,

		stats.AttackPower: 120,
		stats.MeleeCrit:   -0.3 * core.MeleeCritRatingPerCritChance,
	}
}

// Agent is a generic way to access underlying rogue on any of the agents.
type Agent interface {
	GetRogue() *Rogue
}