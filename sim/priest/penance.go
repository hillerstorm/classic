package priest

import (
	"time"

	"github.com/wowsims/sod/sim/core"
	"github.com/wowsims/sod/sim/core/proto"
)

func (priest *Priest) RegisterPenanceSpell() {
	if !priest.HasRune(proto.PriestRune_RuneHandsPenance) {
		return
	}
	priest.Penance = priest.makePenanceSpell(false)
	priest.PenanceHeal = priest.makePenanceSpell(true)
}

// https://www.wowhead.com/classic/spell=402284/penance
// https://www.wowhead.com/classic/news/patch-1-15-build-52124-ptr-datamining-season-of-discovery-runes-336044
func (priest *Priest) makePenanceSpell(isHeal bool) *core.Spell {
	baseDamage := priest.baseRuneAbilityDamage() * 1.28
	baseHealing := priest.baseRuneAbilityDamageHealing() * .85
	spellCoeff := 0.285
	manaCost := .16
	cooldown := time.Second * 12

	var procMask core.ProcMask
	flags := core.SpellFlagChanneled | core.SpellFlagAPL
	if isHeal {
		flags |= core.SpellFlagHelpful
		procMask = core.ProcMaskSpellHealing
	} else {
		procMask = core.ProcMaskSpellDamage
	}

	return priest.RegisterSpell(core.SpellConfig{
		ActionID:      core.ActionID{SpellID: 402284},
		SpellSchool:   core.SpellSchoolHoly,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      procMask,
		Flags:         flags,
		RequiredLevel: 1,

		ManaCost: core.ManaCostOptions{
			BaseCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    priest.NewTimer(),
				Duration: cooldown,
			},
		},

		BonusCritRating: priest.holySpecCritRating() + priest.forceOfWillCritRating(),

		DamageMultiplier: 1,
		ThreatMultiplier: 0,

		Dot: core.Ternary(!isHeal, core.DotConfig{
			Aura: core.Aura{
				Label: "Penance",
			},
			NumberOfTicks:       2,
			TickLength:          time.Second,
			AffectedByCastSpeed: true,

			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dmg := baseDamage + (spellCoeff * dot.Spell.SpellDamage())
				dot.Spell.CalcAndDealPeriodicDamage(sim, target, dmg, dot.OutcomeTick)
			},
		}, core.DotConfig{}),
		Hot: core.Ternary(isHeal, core.DotConfig{
			Aura: core.Aura{
				Label: "Penance",
			},
			NumberOfTicks:       2,
			TickLength:          time.Second,
			AffectedByCastSpeed: true,

			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				healing := baseHealing + spellCoeff*dot.Spell.HealingPower(target)
				dot.Spell.CalcAndDealPeriodicHealing(sim, target, healing, dot.Spell.OutcomeHealingCrit)
			},
		}, core.DotConfig{}),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			if isHeal {
				spell.SpellMetrics[target.UnitIndex].Hits--
				hot := spell.Hot(target)
				hot.Apply(sim)
				// Do immediate tick
				hot.TickOnce(sim)
			} else {
				result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHit)
				if result.Landed() {
					spell.SpellMetrics[target.UnitIndex].Hits--
					dot := spell.Dot(target)
					dot.Apply(sim)
					// Do immediate tick
					dot.TickOnce(sim)
				}
				spell.DealOutcome(sim, result)
			}
		},
	})
}
