package hunter

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

type HunterPet struct {
	core.Pet

	config PetConfig

	hunterOwner *Hunter

	specialAbility *core.Spell
	focusDump      *core.Spell

	uptimePercent    float64
	hasOwnerCooldown bool
}

func (hunter *Hunter) NewHunterPet() *HunterPet {
	if hunter.Options.PetType == proto.Hunter_Options_PetNone {
		return nil
	}
	if hunter.Options.PetUptime <= 0 {
		return nil
	}
	petConfig := PetConfigs[hunter.Options.PetType]

	hunterPetBaseStats := stats.Stats{}

	baseMinDamage := 0.0
	baseMaxDamage := 0.0
	attackSpeed := 2.0

	switch hunter.Options.PetAttackSpeed {
	case proto.Hunter_Options_One:
		attackSpeed = 1.0
	case proto.Hunter_Options_OneTwo:
		attackSpeed = 1.2
	case proto.Hunter_Options_OneThree:
		attackSpeed = 1.3
	case proto.Hunter_Options_OneFour:
		attackSpeed = 1.4
	case proto.Hunter_Options_OneFive:
		attackSpeed = 1.5
	case proto.Hunter_Options_OneSix:
		attackSpeed = 1.6
	case proto.Hunter_Options_OneSeven:
		attackSpeed = 1.7
	case proto.Hunter_Options_Two:
		attackSpeed = 2
	case proto.Hunter_Options_TwoFour:
		attackSpeed = 2.4
	case proto.Hunter_Options_TwoFive:
		attackSpeed = 2.5
	}

	baseMinDamage = 18.5 * attackSpeed
	baseMaxDamage = 28.0 * attackSpeed
	hunterPetBaseStats = stats.Stats{
		stats.Strength:  136,
		stats.Agility:   100,
		stats.Stamina:   274,
		stats.Intellect: 50,
		stats.Spirit:    80,

		stats.AttackPower: -20,
	}

	hp := &HunterPet{
		Pet:         core.NewPet(petConfig.Name, &hunter.Character, hunterPetBaseStats, hunter.makeStatInheritance(), true, false),
		config:      petConfig,
		hunterOwner: hunter,

		hasOwnerCooldown: petConfig.SpecialAbility == FuriousHowl,
	}

	hp.Pet.MobType = petConfig.MobType

	hp.EnableAutoAttacks(hp, core.AutoAttackOptions{
		MainHand: core.Weapon{
			BaseDamageMin: baseMinDamage,
			BaseDamageMax: baseMaxDamage,
			SwingSpeed:    attackSpeed,
		},
		AutoSwingMelee: true,
	})

	// Happiness
	hp.PseudoStats.DamageDealtMultiplier *= 1.25

	// This stuff probably need removedD?
	// Family scalars
	hp.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] *= hp.config.Damage
	hp.PseudoStats.ArmorMultiplier *= hp.config.Armor
	hp.MultiplyStat(stats.Health, hp.config.Health)

	hp.AddStatDependency(stats.Strength, stats.AttackPower, 2)

	// Warrior crit scaling
	hp.AddStatDependency(stats.Agility, stats.MeleeCrit, core.CritPerAgiAtLevel[proto.Class_ClassWarrior]*core.CritRatingPerCritChance)
	hp.AddStatDependency(stats.Intellect, stats.SpellCrit, core.CritPerIntAtLevel[proto.Class_ClassWarrior]*core.SpellCritRatingPerCritChance)

	core.ApplyPetConsumeEffects(&hp.Character, hunter.Consumes)

	hunter.AddPet(hp)

	return hp
}

func (hp *HunterPet) GetPet() *core.Pet {
	return &hp.Pet
}

func (hp *HunterPet) Initialize() {
	hp.specialAbility = hp.NewPetAbility(hp.config.SpecialAbility, true)
	hp.focusDump = hp.NewPetAbility(hp.config.FocusDump, false)

	hp.EnableFocusBar(1, func(sim *core.Simulation) {
		if hp.GCD.IsReady(sim) {
			hp.OnGCDReady(sim)
		}
	})
}

func (hp *HunterPet) Reset(_ *core.Simulation) {
	hp.uptimePercent = min(1, max(0, hp.hunterOwner.Options.PetUptime))
}

func (hp *HunterPet) ExecuteCustomRotation(sim *core.Simulation) {
	percentRemaining := sim.GetRemainingDurationPercent()
	if percentRemaining < 1.0-hp.uptimePercent { // once fight is % completed, disable pet.
		hp.Disable(sim)
		return
	}

	if hp.hasOwnerCooldown && hp.CurrentFocus() < 50 {
		// When a major ability (Furious Howl or Savage Rend) is ready, pool enough
		// energy to use on-demand.
		return
	}

	target := hp.CurrentTarget

	// using Cast() directly is very expensive, since cast failures are logged, involving string operations
	tryCast := func(spell *core.Spell) bool {
		if !spell.CanCast(sim, target) {
			return false
		}
		if !spell.Cast(sim, target) {
			panic(fmt.Sprintf("Cast failed after CanCast() for spell %d", spell.SpellID))
		}
		return true
	}

	if hp.focusDump == nil {
		if !tryCast(hp.specialAbility) && hp.GCD.IsReady(sim) {
			hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
		}
		return
	}
	if hp.specialAbility == nil {
		if !tryCast(hp.focusDump) && hp.GCD.IsReady(sim) {
			hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
		}
		return
	}

	if hp.config.CustomRotation != nil {
		hp.config.CustomRotation(sim, hp, tryCast)
	} else {
		if hp.specialAbility.IsReady(sim) {
			if !tryCast(hp.specialAbility) && hp.GCD.IsReady(sim) {
				hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
			}
		} else if hp.focusDump.IsReady(sim) {
			if !tryCast(hp.focusDump) && hp.GCD.IsReady(sim) {
				hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
			}
		}
	}
}

func (hunter *Hunter) makeStatInheritance() core.PetStatInheritance {
	return func(ownerStats stats.Stats) stats.Stats {
		// EJ posts claim this value is passed through math.Floor, but in-game testing
		// shows pets benefit from each point of owner hit rating in WotLK Classic.
		// https://web.archive.org/web/20120112003252/http://elitistjerks.com/f80/t100099-demonology_releasing_demon_you
		ownerHitChance := ownerStats[stats.MeleeHit] / core.MeleeHitRatingPerHitChance
		hitRatingFromOwner := ownerHitChance * core.MeleeHitRatingPerHitChance

		return stats.Stats{
			stats.Stamina:     ownerStats[stats.Stamina] * 0.3,
			stats.Armor:       ownerStats[stats.Armor] * 0.35,
			stats.AttackPower: ownerStats[stats.RangedAttackPower] * 0.22,

			stats.MeleeCrit: ownerStats[stats.MeleeCrit],
			stats.SpellCrit: ownerStats[stats.MeleeCrit],

			stats.MeleeHit: hitRatingFromOwner,
			stats.SpellHit: hitRatingFromOwner * 2,
		}
	}
}

type PetConfig struct {
	Name    string
	MobType proto.MobType

	SpecialAbility PetAbilityType
	FocusDump      PetAbilityType

	Health float64
	Armor  float64
	Damage float64

	CustomRotation func(*core.Simulation, *HunterPet, func(*core.Spell) bool)
}

// Abilities reference: https://wotlk.wowhead.com/hunter-pets
// https://wotlk.wowhead.com/guides/hunter-dps-best-pets-taming-loyalty-burning-crusade-classic
var PetConfigs = map[proto.Hunter_Options_PetType]PetConfig{
	proto.Hunter_Options_Cat: {
		Name:    "Cat",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Bite,
		FocusDump:      Claw,

		Health: 0.98,
		Armor:  1.00,
		Damage: 1.10,

		CustomRotation: func(sim *core.Simulation, hp *HunterPet, tryCast func(*core.Spell) bool) {
			if hp.specialAbility.CD.IsReady(sim) && hp.CurrentFocusPerSecond() > hp.focusDump.Cost.BaseCost/1.6 {
				if !tryCast(hp.specialAbility) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			} else {
				if !tryCast(hp.focusDump) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			}
		},
	},
	proto.Hunter_Options_WindSerpent: {
		Name:    "Wind Serpent",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Bite,
		FocusDump:      LightningBreath,

		Health: 1.00,
		Armor:  1.00,
		Damage: 1.07,
	},
	proto.Hunter_Options_Bat: {
		Name:    "Bat",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Bite,
		FocusDump:      Screech,

		Health: 1.00,
		Armor:  1.00,
		Damage: 1.07,

		CustomRotation: func(sim *core.Simulation, hp *HunterPet, tryCast func(*core.Spell) bool) {
			if hp.specialAbility.CD.IsReady(sim) {
				if !tryCast(hp.specialAbility) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			} else {
				if !tryCast(hp.focusDump) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			}
		},
	},
	proto.Hunter_Options_Bear: {
		Name:    "Bear",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Bite,
		FocusDump:      Claw,

		Health: 1.08,
		Armor:  1.05,
		Damage: 0.91,
	},
	proto.Hunter_Options_Boar: {
		Name:    "Boar",
		MobType: proto.MobType_MobTypeBeast,

		//SpecialAbility: Gore,
		FocusDump: Bite,

		Health: 1.04,
		Armor:  1.09,
		Damage: 0.90,
	},
	proto.Hunter_Options_CarrionBird: {
		Name:    "Carrion Bird",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Bite, // Screech
		FocusDump:      Claw,

		Health: 1.00,
		Armor:  1.05,
		Damage: 1.00,
	},
	proto.Hunter_Options_Owl: {
		Name:    "Owl",
		MobType: proto.MobType_MobTypeBeast,

		//SpecialAbility: Screech,
		FocusDump: Claw,

		Health: 1.00,
		Armor:  1.00,
		Damage: 1.07,
	},
	proto.Hunter_Options_Crab: {
		Name:    "Crab",
		MobType: proto.MobType_MobTypeBeast,

		FocusDump: Claw,

		Health: 0.96,
		Armor:  1.13,
		Damage: 0.95,
	},
	proto.Hunter_Options_Crocolisk: {
		Name:    "Crocolisk",
		MobType: proto.MobType_MobTypeBeast,

		FocusDump: Bite,

		Health: 0.95,
		Armor:  1.10,
		Damage: 1.00,
	},
	proto.Hunter_Options_Gorilla: {
		Name:    "Gorilla",
		MobType: proto.MobType_MobTypeBeast,

		// SpecialAbility: Thunderstomp,
		FocusDump: Bite,

		Health: 1.04,
		Armor:  1.00,
		Damage: 1.02,
	},
	proto.Hunter_Options_Hyena: {
		Name:    "Hyena",
		MobType: proto.MobType_MobTypeBeast,

		FocusDump: Bite,

		Health: 1.00,
		Armor:  1.05,
		Damage: 1.00,
	},
	proto.Hunter_Options_Raptor: {
		Name:    "Raptor",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Bite,
		FocusDump:      Claw,

		Health: 0.95,
		Armor:  1.03,
		Damage: 1.10,
	},
	proto.Hunter_Options_Scorpid: {
		Name:    "Scorpid",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: ScorpidPoison,
		FocusDump:      Claw,

		Health: 1.00,
		Armor:  1.10,
		Damage: 0.94,

		CustomRotation: func(sim *core.Simulation, hp *HunterPet, tryCast func(*core.Spell) bool) {
			target := hp.CurrentTarget

			if (hp.specialAbility.Dot(target).GetStacks() < hp.specialAbility.Dot(target).MaxStacks || hp.specialAbility.Dot(target).RemainingDuration(sim) < time.Second*3) && hp.CurrentFocus() < 90 {
				if !tryCast(hp.specialAbility) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			} else {
				if !tryCast(hp.focusDump) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			}
		},
	},
	proto.Hunter_Options_Spider: {
		Name:    "Spider",
		MobType: proto.MobType_MobTypeBeast,

		FocusDump: Bite,

		Health: 1.00,
		Armor:  1.00,
		Damage: 1.07,
	},
	proto.Hunter_Options_Tallstrider: {
		Name:    "Tallstrider",
		MobType: proto.MobType_MobTypeBeast,

		FocusDump: Bite,

		Health: 1.05,
		Armor:  1.00,
		Damage: 1.00,
	},
	proto.Hunter_Options_Turtle: {
		Name:    "Turtle",
		MobType: proto.MobType_MobTypeBeast,

		// SpecialAbility: ShellShield,
		FocusDump: Bite,

		Health: 1.00,
		Armor:  1.13,
		Damage: 0.90,
	},
	proto.Hunter_Options_Wolf: {
		Name:    "Wolf",
		MobType: proto.MobType_MobTypeBeast,

		// SpecialAbility: FuriousHowl,
		FocusDump: Bite,

		Health: 1.00,
		Armor:  1.05,
		Damage: 1.00,
	},
}
