// Helpers de texto e DOM.

export const TIPOS_PT = {
  normal: 'Normal', fire: 'Fogo', water: 'Água', electric: 'Elétrico',
  grass: 'Planta', ice: 'Gelo', fighting: 'Lutador', poison: 'Venenoso',
  ground: 'Terra', flying: 'Voador', psychic: 'Psíquico', bug: 'Inseto',
  rock: 'Pedra', ghost: 'Fantasma', dragon: 'Dragão', dark: 'Sombrio',
  steel: 'Aço', fairy: 'Fada',
};

export const TIERS = ['lendario', 'epico', 'raro', 'comum'];
export const TIER_PT = { comum: 'Comum', raro: 'Raro', epico: 'Épico', lendario: 'Lendário' };
export const TIER_ESTRELAS = { comum: '', raro: '★', epico: '★★', lendario: '★★★' };

export const traduzirTipo = (t) => TIPOS_PT[t] ?? t;

// Mesma regra do back-end: "flutter-mane" -> "Flutter Mane".
export function exibir(nome) {
  if (!nome) return '-';
  return nome.split('-').map((p) => (p ? p[0].toUpperCase() + p.slice(1) : p)).join(' ');
}

export function escapar(texto) {
  return String(texto ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

// Cria elemento a partir de HTML (o chamador escapa os dados com escapar()).
export function html(texto) {
  const t = document.createElement('template');
  t.innerHTML = texto.trim();
  return t.content.firstElementChild;
}

export const espera = (ms) => new Promise((r) => setTimeout(r, ms));

// Reinicia uma animação CSS que pode estar no meio.
export function animar(el, classe, duracaoMs) {
  if (!el) return Promise.resolve();
  el.classList.remove(classe);
  void el.offsetWidth;
  el.classList.add(classe);
  return espera(duracaoMs).then(() => el.classList.remove(classe));
}

export function sinal(n) { return n > 0 ? `+${n}` : String(n); }
