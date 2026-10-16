// roteador por hash: #login, #menu, #gacha, #colecao, #time, #batalha.
import { Estado } from './state.js';

const telas = {};
const publicas = new Set(['login']);
let limparAtual = null;

export function registrar(nome, render) { telas[nome] = render; }

export function navegar(tela) {
  if (location.hash.slice(1) === tela) renderizar();
  else location.hash = tela;
}

async function renderizar() {
  let tela = location.hash.slice(1) || 'login';
  if (!telas[tela]) tela = 'login';
  if (!publicas.has(tela) && !Estado.jogador) tela = 'login';

  if (typeof limparAtual === 'function') limparAtual();
  limparAtual = null;

  const app = document.getElementById('app');
  app.innerHTML = '';
  app.className = `tela tela-${tela}`;
  window.scrollTo(0, 0);
  limparAtual = await telas[tela](app);
}

export function iniciarRoteador() {
  window.addEventListener('hashchange', renderizar);
  renderizar();
}
