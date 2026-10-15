import { traduzirTipo, escapar } from '../utils.js';

export function tipoBadge(tipo) {
  return `<span class="tipo-badge" style="--cor-tipo: var(--type-${escapar(tipo)})">${escapar(traduzirTipo(tipo))}</span>`;
}

export function tiposHTML(tipos = []) {
  return `<span class="tipos">${tipos.map(tipoBadge).join('')}</span>`;
}
