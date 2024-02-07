package sod

import (
	"github.com/wowsims/sod/sim/core"
	"github.com/wowsims/sod/sim/core/proto"
	"github.com/wowsims/sod/sim/core/stats"
)

// Keep these in alphabetical order.

var ItemSetStormshroud = core.NewItemSet(core.ItemSet{
	Name: "Stormshroud Armor",
	Bonuses: map[int32]core.ApplyEffect{
		2: func(a core.Agent) {
			char := a.GetCharacter()
			proc := char.RegisterSpell(core.SpellConfig{
				ActionID:    core.ActionID{SpellID: 18980},
				SpellSchool: core.SpellSchoolNature,
				ProcMask:    core.ProcMaskEmpty,

				DamageMultiplier: 1,
				CritMultiplier:   char.DefaultSpellCritMultiplier(),
				ThreatMultiplier: 1,

				ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
					spell.CalcAndDealDamage(sim, target, sim.Roll(15, 25), spell.OutcomeMagicHitAndCrit)
				},
			})
			char.RegisterAura(core.Aura{
				Label:    "Stormshround Armor 2pc",
				ActionID: core.ActionID{SpellID: 18979},
				Duration: core.NeverExpires,
				OnReset: func(aura *core.Aura, sim *core.Simulation) {
					aura.Activate(sim)
				},
				OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					if !result.Landed() || !spell.ProcMask.Matches(core.ProcMaskMelee) {
						return
					}
					if sim.RandomFloat("Stormshroud Armor 2pc") < 0.05 {
						proc.Cast(sim, result.Target)
					}
				},
			})
		},
		3: func(a core.Agent) {
			char := a.GetCharacter()
			if !char.HasEnergyBar() {
				return
			}
			metrics := char.NewEnergyMetrics(core.ActionID{SpellID: 23863})
			proc := char.RegisterSpell(core.SpellConfig{
				ActionID:    core.ActionID{SpellID: 23864},
				SpellSchool: core.SpellSchoolNature,
				ApplyEffects: func(sim *core.Simulation, u *core.Unit, spell *core.Spell) {
					char.AddEnergy(sim, 30, metrics)
				},
			})
			char.RegisterAura(core.Aura{
				Label:    "Stormshround Armor 3pc",
				ActionID: core.ActionID{SpellID: 18979},
				Duration: core.NeverExpires,
				OnReset: func(aura *core.Aura, sim *core.Simulation) {
					aura.Activate(sim)
				},
				OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					if !result.Landed() || !spell.ProcMask.Matches(core.ProcMaskMelee) {
						return
					}
					if sim.RandomFloat("Stormshroud Armor 3pc") < 0.02 {
						proc.Cast(sim, result.Target)
					}
				},
			})

		},
		4: func(a core.Agent) {
			a.GetCharacter().AddStat(stats.AttackPower, 14)
		},
	},
})

var ItemSetBlackfathomAvengerMail = core.NewItemSet(core.ItemSet{
	Name: "Blackfathom Avenger's Mail",
	Bonuses: map[int32]core.ApplyEffect{
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.AttackPower, 12)
		},
		3: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.MeleeHit, 1)
			c.AddStat(stats.SpellHit, 1)
		},
	},
})

var ItemSetBlackfathomSlayerLeather = core.NewItemSet(core.ItemSet{
	Name: "Blackfathom Slayer's Leather",
	Bonuses: map[int32]core.ApplyEffect{
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.AttackPower, 12)
		},
		3: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.MeleeHit, 1)
			c.AddStat(stats.SpellHit, 1)
		},
	},
})

var ItemSetElectromanticDevastator = core.NewItemSet(core.ItemSet{
	Name: "Electromantic Devastator's Mail",
	Bonuses: map[int32]core.ApplyEffect{
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.AttackPower, 24)
		},
		3: func(agent core.Agent) {
			c := agent.GetCharacter()
			if !c.HasManaBar() {
				return
			}
			metrics := c.NewManaMetrics(core.ActionID{SpellID: 435982})
			proc := c.RegisterSpell(core.SpellConfig{
				ActionID:    core.ActionID{SpellID: 435981},
				SpellSchool: core.SpellSchoolHoly,
				ApplyEffects: func(sim *core.Simulation, u *core.Unit, spell *core.Spell) {
					c.AddMana(sim, 100, metrics)
				},
			})
			procChance := 0.05
			c.RegisterAura(core.Aura{
				Label:    "Electromantic Devastator's Mail 3pc",
				ActionID: core.ActionID{SpellID: 435982},
				Duration: core.NeverExpires,
				OnReset: func(aura *core.Aura, sim *core.Simulation) {
					aura.Activate(sim)
				},
				// Modeled after WotLK JoW https://github.com/wowsims/wotlk/blob/master/sim/core/debuffs.go#L202
				OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					if spell.ProcMask.Matches(core.ProcMaskEmpty | core.ProcMaskProc | core.ProcMaskWeaponProc) {
						return // Phantom spells don't proc
					}

					if spell.ProcMask.Matches(core.ProcMaskWhiteHit | core.ProcMaskRanged) { // Ranged/melee can proc on miss
						if sim.RandomFloat("Electromantic Devastator's Mail 3pc") > procChance {
							return
						}
					} else { // Spell Casting only procs on hits
						if !result.Landed() {
							return
						}
						if sim.RandomFloat("Electromantic Devastator's Mail 3pc") > procChance {
							return
						}
					}
					proc.Cast(sim, result.Target)
				},
			})
		},
	},
})

var ItemSetInsulatedLeather = core.NewItemSet(core.ItemSet{
	Name: "Insulated Leathers",
	Bonuses: map[int32]core.ApplyEffect{
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.MeleeCrit, 1)
			c.AddStat(stats.SpellCrit, 1)
		},
		// TODO: Implement Feral set bonus
		3: func(agent core.Agent) {
			c := agent.GetCharacter()
			applyWeaponSpecialization(c, 3, proto.WeaponType_WeaponTypeDagger)
		},
	},
})

var ItemSetHazardSuit = core.NewItemSet(core.ItemSet{
	Name: "H.A.Z.A.R.D. Suit",
	Bonuses: map[int32]core.ApplyEffect{
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.Defense, 7)
			c.AddStat(stats.AttackPower, 16)
		},
		3: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.MeleeHit, 1)
			c.AddStat(stats.SpellHit, 1)
		},
	},
})

func applyWeaponSpecialization(character *core.Character, weaponSkillBonus float64, weaponTypes ...proto.WeaponType) {
	mask := character.GetProcMaskForTypes(weaponTypes...)

	if mask == core.ProcMaskMelee || (mask == core.ProcMaskMeleeMH && !character.HasOHWeapon()) {
		character.AddStat(stats.WeaponSkill, weaponSkillBonus)
	}
}
