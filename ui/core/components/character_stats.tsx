import tippy from 'tippy.js';
import { ref } from 'tsx-vanilla';

import * as Mechanics from '../constants/mechanics.js';
import { Player } from '../player.js';
import { HandType, ItemSlot, PseudoStat, Spec, Stat, WeaponType } from '../proto/common.js';
import { Stats, UnitStat } from '../proto_utils/stats.js';
import { EventID, TypedEvent } from '../typed_event.js';
import { Component } from './component.js';
import { NumberPicker } from './number_picker';

export type StatMods = { talents?: Stats; buffs?: Stats };

const statGroups = new Map<string, Array<UnitStat>>([
	['Primary', [UnitStat.fromStat(Stat.StatHealth), UnitStat.fromStat(Stat.StatMana)]],
	[
		'Attributes',
		[
			UnitStat.fromStat(Stat.StatStrength),
			UnitStat.fromStat(Stat.StatAgility),
			UnitStat.fromStat(Stat.StatStamina),
			UnitStat.fromStat(Stat.StatIntellect),
			UnitStat.fromStat(Stat.StatSpirit),
		],
	],
	[
		'Physical',
		[
			UnitStat.fromStat(Stat.StatAttackPower),
			UnitStat.fromStat(Stat.StatFeralAttackPower),
			UnitStat.fromStat(Stat.StatRangedAttackPower),
			UnitStat.fromStat(Stat.StatMeleeHit),
			UnitStat.fromStat(Stat.StatExpertise),
			UnitStat.fromStat(Stat.StatMeleeCrit),
			UnitStat.fromStat(Stat.StatMeleeHaste),
			UnitStat.fromPseudoStat(PseudoStat.BonusPhysicalDamage),
		],
	],
	[
		'Spell',
		[
			UnitStat.fromStat(Stat.StatSpellPower),
			UnitStat.fromStat(Stat.StatSpellDamage),
			UnitStat.fromStat(Stat.StatArcanePower),
			UnitStat.fromStat(Stat.StatFirePower),
			UnitStat.fromStat(Stat.StatFrostPower),
			UnitStat.fromStat(Stat.StatHolyPower),
			UnitStat.fromStat(Stat.StatNaturePower),
			UnitStat.fromStat(Stat.StatShadowPower),
			UnitStat.fromStat(Stat.StatSpellHit),
			UnitStat.fromStat(Stat.StatSpellCrit),
			UnitStat.fromStat(Stat.StatSpellHaste),
			UnitStat.fromStat(Stat.StatSpellPenetration),
			UnitStat.fromStat(Stat.StatMP5),
		],
	],
	[
		'Defense',
		[
			UnitStat.fromStat(Stat.StatArmor),
			UnitStat.fromStat(Stat.StatBonusArmor),
			UnitStat.fromStat(Stat.StatDefense),
			UnitStat.fromStat(Stat.StatDodge),
			UnitStat.fromStat(Stat.StatParry),
			UnitStat.fromStat(Stat.StatBlock),
			UnitStat.fromStat(Stat.StatBlockValue),
		],
	],
	[
		'Resistance',
		[
			UnitStat.fromStat(Stat.StatArcaneResistance),
			UnitStat.fromStat(Stat.StatFireResistance),
			UnitStat.fromStat(Stat.StatFrostResistance),
			UnitStat.fromStat(Stat.StatNatureResistance),
			UnitStat.fromStat(Stat.StatShadowResistance),
		],
	],
	['Misc', []],
]);

export class CharacterStats extends Component {
	readonly stats: Array<UnitStat>;
	readonly valueElems: Array<HTMLTableCellElement>;
	readonly meleeCritCapValueElem: HTMLTableCellElement | undefined;

	private readonly player: Player<any>;
	private readonly modifyDisplayStats?: (player: Player<any>) => StatMods;

	constructor(parent: HTMLElement, player: Player<any>, displayStats: Array<UnitStat>, modifyDisplayStats?: (player: Player<any>) => StatMods) {
		super(parent, 'character-stats-root');
		this.stats = [];
		this.player = player;
		this.modifyDisplayStats = modifyDisplayStats;

		const table = <table className="character-stats-table"></table>;
		this.rootElem.appendChild(table);

		this.valueElems = [];
		statGroups.forEach((groupedStats, _) => {
			const filteredStats = groupedStats.filter(stat => displayStats.find(displayStat => displayStat.equals(stat)));

			if (!filteredStats.length) return;

			const body = <tbody></tbody>;
			filteredStats.forEach(stat => {
				this.stats.push(stat);

				const statName = stat.getName(player.getClass());

				const row = (
					<tr className="character-stats-table-row">
						<td className="character-stats-table-label">{statName}</td>
						<td className="character-stats-table-value">{this.bonusStatsLink(stat)}</td>
					</tr>
				);
				body.appendChild(row);

				const valueElem = row.getElementsByClassName('character-stats-table-value')[0] as HTMLTableCellElement;
				this.valueElems.push(valueElem);
			});

			table.appendChild(body);
		});

		if (this.shouldShowMeleeCritCap(player)) {
			const row = (
				<tr className="character-stats-table-row">
					<td className="character-stats-table-label">Melee Crit Cap</td>
					<td className="character-stats-table-value"></td>
				</tr>
			);

			table.appendChild(row);
			this.meleeCritCapValueElem = row.getElementsByClassName('character-stats-table-value')[0] as HTMLTableCellElement;
		}

		this.updateStats(player);
		TypedEvent.onAny([player.currentStatsEmitter, player.sim.changeEmitter, player.talentsChangeEmitter]).on(() => {
			this.updateStats(player);
		});
	}

	private updateStats(player: Player<any>) {
		const playerStats = player.getCurrentStats();

		const statMods = this.modifyDisplayStats ? this.modifyDisplayStats(this.player) : {};
		if (!statMods.talents) statMods.talents = new Stats();
		if (!statMods.buffs) statMods.buffs = new Stats();

		const baseStats = Stats.fromProto(playerStats.baseStats);
		const gearStats = Stats.fromProto(playerStats.gearStats);
		const talentsStats = Stats.fromProto(playerStats.talentsStats);
		const buffsStats = Stats.fromProto(playerStats.buffsStats);
		const consumesStats = Stats.fromProto(playerStats.consumesStats);
		const debuffStats = this.getDebuffStats();
		const bonusStats = player.getBonusStats();

		const baseDelta = baseStats;
		const gearDelta = gearStats.subtract(baseStats).subtract(bonusStats);
		const talentsDelta = talentsStats.subtract(gearStats).add(statMods.talents);
		const buffsDelta = buffsStats.subtract(talentsStats).add(statMods.buffs);
		const consumesDelta = consumesStats.subtract(buffsStats);

		const finalStats = Stats.fromProto(playerStats.finalStats).add(statMods.talents).add(statMods.buffs).add(debuffStats);

		this.stats.forEach((stat, idx) => {
			const bonusStatValue = bonusStats.getUnitStat(stat);
			let contextualClass: string;
			if (bonusStatValue === 0) {
				contextualClass = 'text-white';
			} else if (bonusStatValue > 0) {
				contextualClass = 'text-success';
			} else {
				contextualClass = 'text-danger';
			}

			const statLinkElemRef = ref<HTMLAnchorElement>();

			const valueElem = (
				<div className="stat-value-link-container">
					<a href="javascript:void(0)" className={`stat-value-link ${contextualClass}`} attributes={{ role: 'button' }} ref={statLinkElemRef}>
						{`${this.statDisplayString(player, finalStats, finalStats, stat)} `}
					</a>
				</div>
			);

			const statLinkElem = statLinkElemRef.value!;

			this.valueElems[idx].querySelector('.stat-value-link-container')?.remove();
			this.valueElems[idx].prepend(valueElem);

			const tooltipContent = (
				<div className="d-flex">
					<div>
						<div className="character-stats-tooltip-row">
							<span>Base:</span>
							<span>{this.statDisplayString(player, baseStats, baseDelta, stat)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>Gear:</span>
							<span>{this.statDisplayString(player, gearStats, gearDelta, stat)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>Talents:</span>
							<span>{this.statDisplayString(player, talentsStats, talentsDelta, stat)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>Buffs:</span>
							<span>{this.statDisplayString(player, buffsStats, buffsDelta, stat)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>Consumes:</span>
							<span>{this.statDisplayString(player, consumesStats, consumesDelta, stat)}</span>
						</div>
						{stat.isStat() && debuffStats.getStat(stat.getStat()) != 0 && (
							<div className="character-stats-tooltip-row">
								<span>Debuffs:</span>
								<span>{this.statDisplayString(player, debuffStats, debuffStats, stat)}</span>
							</div>
						)}
						{bonusStatValue != 0 && (
							<div className="character-stats-tooltip-row">
								<span>Bonus:</span>
								<span>{this.statDisplayString(player, bonusStats, bonusStats, stat)}</span>
							</div>
						)}
						<div className="character-stats-tooltip-row">
							<span>Total:</span>
							<span>{this.statDisplayString(player, finalStats, finalStats, stat)}</span>
						</div>
					</div>
				</div>
			);

			if (stat.isStat() && stat.getStat() === Stat.StatMeleeHit) {
				tooltipContent.appendChild(
					<div className="ps-2">
						<div className="character-stats-tooltip-row">
							<span>Axes</span>
							<span>{this.weaponSkillDisplayString(talentsStats, PseudoStat.PseudoStatAxesSkill)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>2H Axes</span>
							<span>{this.weaponSkillDisplayString(talentsStats, PseudoStat.PseudoStatTwoHandedAxesSkill)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>Daggers</span>
							<span>{this.weaponSkillDisplayString(talentsStats, PseudoStat.PseudoStatDaggersSkill)}</span>
						</div>
						{/*  Commenting out feral combat skill since not present in Classic.
						{player.spec === Spec.SpecFeralDruid && (
							<div className="character-stats-tooltip-row">
								<span>Feral Combat</span>
								<span>{this.weaponSkillDisplayString(talentsStats, PseudoStat.PseudoStatFeralCombatSkill)}</span>
							</div>
						)}
						*/}
						<div className="character-stats-tooltip-row">
							<span>Maces</span>
							<span>{this.weaponSkillDisplayString(talentsStats, PseudoStat.PseudoStatMacesSkill)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>2H Maces</span>
							<span>{this.weaponSkillDisplayString(talentsStats, PseudoStat.PseudoStatTwoHandedMacesSkill)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>Polearms</span>
							<span>{this.weaponSkillDisplayString(talentsStats, PseudoStat.PseudoStatPolearmsSkill)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>Staves</span>
							<span>{this.weaponSkillDisplayString(talentsStats, PseudoStat.PseudoStatStavesSkill)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>Swords</span>
							<span>{this.weaponSkillDisplayString(talentsStats, PseudoStat.PseudoStatSwordsSkill)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>2H Swords</span>
							<span>{this.weaponSkillDisplayString(talentsStats, PseudoStat.PseudoStatTwoHandedSwordsSkill)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>Unarmed</span>
							<span>{this.weaponSkillDisplayString(talentsStats, PseudoStat.PseudoStatUnarmedSkill)}</span>
						</div>
					</div>,
				);
			} else if (stat.isStat() && stat.getStat() === Stat.StatSpellHit) {
				tooltipContent.appendChild(
					<div className="ps-2">
						<div className="character-stats-tooltip-row">
							<span>Arcane</span>
							<span>{this.spellSchoolHitDisplayString(finalStats, PseudoStat.PseudoStatSchoolHitArcane)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>Fire</span>
							<span>{this.spellSchoolHitDisplayString(finalStats, PseudoStat.PseudoStatSchoolHitFire)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>Frost</span>
							<span>{this.spellSchoolHitDisplayString(finalStats, PseudoStat.PseudoStatSchoolHitFrost)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>Holy</span>
							<span>{this.spellSchoolHitDisplayString(finalStats, PseudoStat.PseudoStatSchoolHitHoly)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>Nature</span>
							<span>{this.spellSchoolHitDisplayString(finalStats, PseudoStat.PseudoStatSchoolHitNature)}</span>
						</div>
						<div className="character-stats-tooltip-row">
							<span>Shadow</span>
							<span>{this.spellSchoolHitDisplayString(finalStats, PseudoStat.PseudoStatSchoolHitShadow)}</span>
						</div>
					</div>,
				);
			}

			tippy(statLinkElem, {
				content: tooltipContent,
			});
		});

		if (this.meleeCritCapValueElem) {
			const has2hWeapon = player.getGear().getEquippedItem(ItemSlot.ItemSlotMainHand)?.item.handType == HandType.HandTypeTwoHand;
			const mhWeapon: WeaponType = player.getGear().getEquippedItem(ItemSlot.ItemSlotMainHand)?.item.weaponType as WeaponType;
			const ohWeapon: WeaponType = player.getGear().getEquippedItem(ItemSlot.ItemSlotOffHand)?.item.weaponType as WeaponType;
			const mhCritCapInfo = player.getMeleeCritCapInfo(mhWeapon, has2hWeapon);
			const ohCritCapInfo = player.getMeleeCritCapInfo(ohWeapon, has2hWeapon);

			const playerCritCapDelta = mhCritCapInfo.playerCritCapDelta;

			if (playerCritCapDelta === 0.0) {
				const prefix = 'Exact';
			}

			const prefix = playerCritCapDelta > 0 ? 'Over by ' : 'Under by ';

			const valueElem = (
				<a href="javascript:void(0)" className="stat-value-link" attributes={{ role: 'button' }}>
					{`${prefix} ${Math.abs(playerCritCapDelta).toFixed(2)}%`}
				</a>
			);

			const capDelta = mhCritCapInfo.playerCritCapDelta;
			if (capDelta === 0) {
				valueElem.classList.add('text-white');
			} else if (capDelta > 0) {
				valueElem.classList.add('text-danger');
			} else if (capDelta < 0) {
				valueElem.classList.add('text-success');
			}

			this.meleeCritCapValueElem.querySelector('.stat-value-link')?.remove();
			this.meleeCritCapValueElem.prepend(valueElem);

			const tooltipContent = (
				<div>
					<div className="character-stats-tooltip-row">
						<span>Main Hand</span>
						<span></span>
					</div>
					<hr />
					<div className="character-stats-tooltip-row">
						<span>Glancing:</span>
						<span>{`${mhCritCapInfo.glancing.toFixed(2)}%`}</span>
					</div>
					<div className="character-stats-tooltip-row">
						<span>Suppression:</span>
						<span>{`${mhCritCapInfo.suppression.toFixed(2)}%`}</span>
					</div>
					<div className="character-stats-tooltip-row">
						<span>White Miss:</span>
						<span>{`${mhCritCapInfo.remainingMeleeHitCap.toFixed(2)}%`}</span>
					</div>
					<div className="character-stats-tooltip-row">
						<span>Dodge:</span>
						<span>{`${mhCritCapInfo.dodgeCap.toFixed(2)}%`}</span>
					</div>
					<div className="character-stats-tooltip-row">
						<span>Parry:</span>
						<span>{`${mhCritCapInfo.parryCap.toFixed(2)}%`}</span>
					</div>
					{mhCritCapInfo.specSpecificOffset != 0 && (
						<div className="character-stats-tooltip-row">
							<span>Spec Offsets:</span>
							<span>{`${mhCritCapInfo.specSpecificOffset.toFixed(2)}%`}</span>
						</div>
					)}
					<div className="character-stats-tooltip-row">
						<span>Final Crit Cap:</span>
						<span>{`${mhCritCapInfo.baseCritCap.toFixed(2)}%`}</span>
					</div>
					<div className="character-stats-tooltip-row">
						<span>Can Raise By:</span>
						<span>{`${mhCritCapInfo.remainingMeleeHitCap.toFixed(2)}%`}</span>
					</div>
					{!has2hWeapon && (
						<div>
							<hr />

							<div className="character-stats-tooltip-row">
								<span>Off Hand</span>
								<span></span>
							</div>
							<hr />
							<div className="character-stats-tooltip-row">
								<span>Glancing:</span>
								<span>{`${ohCritCapInfo.glancing.toFixed(2)}%`}</span>
							</div>
							<div className="character-stats-tooltip-row">
								<span>Suppression:</span>
								<span>{`${ohCritCapInfo.suppression.toFixed(2)}%`}</span>
							</div>
							<div className="character-stats-tooltip-row">
								<span>White Miss:</span>
								<span>{`${ohCritCapInfo.remainingMeleeHitCap.toFixed(2)}%`}</span>
							</div>
							<div className="character-stats-tooltip-row">
								<span>Dodge:</span>
								<span>{`${ohCritCapInfo.dodgeCap.toFixed(2)}%`}</span>
							</div>
							<div className="character-stats-tooltip-row">
								<span>Parry:</span>
								<span>{`${ohCritCapInfo.parryCap.toFixed(2)}%`}</span>
							</div>
							{ohCritCapInfo.specSpecificOffset != 0 && (
								<div className="character-stats-tooltip-row">
									<span>Spec Offsets:</span>
									<span>{`${ohCritCapInfo.specSpecificOffset.toFixed(2)}%`}</span>
								</div>
							)}
							<div className="character-stats-tooltip-row">
								<span>Final Crit Cap:</span>
								<span>{`${ohCritCapInfo.baseCritCap.toFixed(2)}%`}</span>
							</div>
							<div className="character-stats-tooltip-row">
								<span>Can Raise By:</span>
								<span>{`${ohCritCapInfo.remainingMeleeHitCap.toFixed(2)}%`}</span>
							</div>
						</div>
					)}
				</div>
			);

			tippy(valueElem, {
				content: tooltipContent,
			});
		}
	}

	private statDisplayString(player: Player<any>, stats: Stats, deltaStats: Stats, unitStat: UnitStat): string {
		const rawValue = deltaStats.getUnitStat(unitStat);
		let displayStr: string | undefined;

		if (unitStat.isStat()) {
			const stat = unitStat.getStat();

			if (stat === Stat.StatBlockValue) {
				const mult = stats.getPseudoStat(PseudoStat.PseudoStatBlockValueMultiplier) || 1;
				const perStr = Math.max(0, stats.getPseudoStat(PseudoStat.PseudoStatBlockValuePerStrength) * deltaStats.getStat(Stat.StatStrength) - 1);
				displayStr = String(Math.round(rawValue * mult + perStr));
			} else if (stat === Stat.StatMeleeHit) {
				displayStr = `${(rawValue / Mechanics.MELEE_HIT_RATING_PER_HIT_CHANCE).toFixed(2)}%`;
			} else if (stat === Stat.StatSpellHit) {
				displayStr = `${(rawValue / Mechanics.SPELL_HIT_RATING_PER_HIT_CHANCE).toFixed(2)}%`;
			} else if (stat === Stat.StatSpellDamage) {
				const spDmg = Math.round(rawValue);
				const baseSp = Math.round(deltaStats.getStat(Stat.StatSpellPower));
				displayStr = baseSp + spDmg + ` (+${spDmg})`;
			} else if (
				stat === Stat.StatArcanePower ||
				stat === Stat.StatFirePower ||
				stat === Stat.StatFrostPower ||
				stat === Stat.StatHolyPower ||
				stat === Stat.StatNaturePower ||
				stat === Stat.StatShadowPower
			) {
				const spDmg = Math.round(rawValue);
				const baseSp = Math.round(deltaStats.getStat(Stat.StatSpellPower) + deltaStats.getStat(Stat.StatSpellDamage));
				displayStr = baseSp + spDmg + ` (+${spDmg})`;
			} else if (stat === Stat.StatMeleeCrit || stat === Stat.StatSpellCrit) {
				displayStr = `${(rawValue / Mechanics.SPELL_CRIT_RATING_PER_CRIT_CHANCE).toFixed(2)}%`;
			} else if (stat === Stat.StatMeleeHaste) {
				// Melee Haste doesn't actually exist in vanilla so use the melee speed pseudostat
				displayStr = `${(deltaStats.getPseudoStat(PseudoStat.PseudoStatMeleeSpeedMultiplier) * 100).toFixed(2)}%`;
			} else if (stat === Stat.StatSpellHaste) {
				displayStr = `${(rawValue / Mechanics.HASTE_RATING_PER_HASTE_PERCENT).toFixed(2)}%`;
			} else if (stat === Stat.StatArmorPenetration) {
				displayStr = `${rawValue} (${(rawValue / Mechanics.ARMOR_PEN_PER_PERCENT_ARMOR).toFixed(2)}%)`;
			} else if (stat === Stat.StatExpertise) {
				// It's just like crit and hit in SoD.
				displayStr = `${rawValue}%`;
			} else if (stat === Stat.StatDefense) {
				displayStr = `${(Mechanics.MAX_CHARACTER_LEVEL * 5 + Math.floor(rawValue / Mechanics.DEFENSE_RATING_PER_DEFENSE)).toFixed(0)}`;
			} else if (stat === Stat.StatBlock) {
				displayStr = `${(rawValue / Mechanics.BLOCK_RATING_PER_BLOCK_CHANCE).toFixed(2)}%`;
			} else if (stat === Stat.StatDodge) {
				displayStr = `${(rawValue / Mechanics.DODGE_RATING_PER_DODGE_CHANCE).toFixed(2)}%`;
			} else if (stat === Stat.StatParry) {
				displayStr = `${(rawValue / Mechanics.PARRY_RATING_PER_PARRY_CHANCE).toFixed(2)}%`;
			} else if (stat === Stat.StatResilience) {
				displayStr = `${rawValue} (${(rawValue / Mechanics.RESILIENCE_RATING_PER_CRIT_REDUCTION_CHANCE).toFixed(2)}%)`;
			}
		}

		if (!displayStr) displayStr = String(Math.round(rawValue));

		return displayStr;
	}

	private weaponSkillDisplayString(stats: Stats, pseudoStat: PseudoStat): string {
		return `${300 + stats.getPseudoStat(pseudoStat)}`;
	}

	private spellSchoolHitDisplayString(stats: Stats, pseudoStat: PseudoStat): string {
		return `${(stats.getPseudoStat(pseudoStat) + stats.getStat(Stat.StatSpellHit)).toFixed(2)}%`;
	}

	private getDebuffStats(): Stats {
		const debuffStats = new Stats();

		// TODO: Classic ui debuffs
		// const debuffs = this.player.sim.raid.getDebuffs();
		// if (debuffs.improvedScorch || debuffs.wintersChill || debuffs.shadowMastery) {
		// 	debuffStats = debuffStats.addStat(Stat.StatSpellCrit, 5 * Mechanics.SPELL_CRIT_RATING_PER_CRIT_CHANCE);
		// }

		return debuffStats;
	}

	private bonusStatsLink(stat: UnitStat): HTMLElement {
		const statName = stat.getName(this.player.getClass());
		const linkRef = ref<HTMLAnchorElement>();
		const iconRef = ref<HTMLDivElement>();

		const link = (
			<a
				ref={linkRef}
				href="javascript:void(0)"
				className="add-bonus-stats text-white ms-2"
				dataset={{ bsToggle: 'popover' }}
				attributes={{ role: 'button' }}>
				<i ref={iconRef} className="fas fa-plus-minus"></i>
			</a>
		);

		tippy(iconRef.value!, { content: `Bonus ${statName}` });
		tippy(linkRef.value!, {
			interactive: true,
			trigger: 'click',
			theme: 'bonus-stats-popover',
			placement: 'right',
			onShow: instance => {
				const picker = new NumberPicker(null, this.player, {
					id: `character-bonus-${stat.isStat() ? 'stat-' + stat.getStat() : 'pseudostat-' + stat.getPseudoStat()}`,
					label: `Bonus ${statName}`,
					extraCssClasses: ['mb-0'],
					changedEvent: (player: Player<any>) => player.bonusStatsChangeEmitter,
					getValue: (player: Player<any>) => player.getBonusStats().getUnitStat(stat),
					setValue: (eventID: EventID, player: Player<any>, newValue: number) => {
						const bonusStats = player.getBonusStats().withUnitStat(stat, newValue);
						player.setBonusStats(eventID, bonusStats);
						instance?.hide();
					},
				});
				instance.setContent(picker.rootElem);
			},
		});

		return link as HTMLElement;
	}

	private shouldShowMeleeCritCap(player: Player<any>): boolean {
		return [Spec.SpecEnhancementShaman, Spec.SpecRetributionPaladin, Spec.SpecRogue, Spec.SpecWarrior, Spec.SpecHunter].includes(player.spec);
	}
}
