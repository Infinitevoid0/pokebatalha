import { TIER_PT, TIER_ESTRELAS, escapar } from '../utils.js';

export function tierBadge(tier, { estrelas = true } = {}) {
  const t = TIER_PT[tier] ? tier : 'comum';
  const prefixo = estrelas && TIER_ESTRELAS[t] ? `${TIER_ESTRELAS[t]} ` : '';
  return `<span class="tier-badge tier-${t}">${escapar(prefixo + TIER_PT[t])}</span>`;
}
