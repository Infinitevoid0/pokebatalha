// diálogo estilo GBA: texto letra a letra e seta "▼".
import { escapar, espera } from '../utils.js';

const MS_POR_LETRA = 22;

// digita o texto no elemento; clicar pula a animação.
export async function digitar(el, texto, { msPorLetra = MS_POR_LETRA } = {}) {
  let pular = false;
  const pularFn = () => { pular = true; };
  el.addEventListener('click', pularFn, { once: true });
  el.textContent = '';
  for (let i = 0; i < texto.length; i++) {
    if (pular) { el.textContent = texto; break; }
    el.textContent += texto[i];
    if (texto[i] !== ' ') await espera(msPorLetra);
  }
  el.removeEventListener('click', pularFn);
}

export function criarDialogo(texto = '') {
  const el = document.createElement('div');
  el.className = 'dialogo';
  el.innerHTML = `<p class="dialogo-texto">${escapar(texto)}</p><span class="seta" aria-hidden="true">▼</span>`;
  return el;
}

// aviso flutuante global; some ao clicar, com Esc ou sozinho.
let atual = null;
export const DialogBox = {
  async mostrar(texto, variante = 'info', duracaoMs = 4500) {
    const raiz = document.getElementById('dialog-root');
    this.fechar();
    const el = criarDialogo('');
    el.classList.add('dialogo-flutuante', variante);
    el.setAttribute('role', variante === 'erro' ? 'alert' : 'status');
    el.tabIndex = 0;
    raiz.appendChild(el);
    atual = el;
    if (variante === 'erro') el.classList.add('anim-shake');

    const fechar = () => this.fechar(el);
    el.addEventListener('click', fechar);
    const timer = setTimeout(fechar, duracaoMs);
    el._timer = timer;
    await digitar(el.querySelector('.dialogo-texto'), texto, { msPorLetra: 12 });
  },
  fechar(el = atual) {
    if (!el) return;
    clearTimeout(el._timer);
    el.remove();
    if (el === atual) atual = null;
  },
};

document.addEventListener('keydown', (e) => { if (e.key === 'Escape') DialogBox.fechar(); });

// mensagens amigáveis por código de erro da API.
const MENSAGENS = {
  jogador_nao_encontrado: 'Jogador não encontrado. Verifique o ID.',
  pokemon_nao_encontrado: 'Pokémon não encontrado.',
  nome_em_uso: 'Esse nome já está em uso. Escolha outro.',
  compra_simultanea: 'Aguarde, outro pacote está sendo aberto.',
  saldo_insuficiente: 'Moedas insuficientes! Vença batalhas para ganhar mais.',
  servico_externo_indisponivel: 'A PokeAPI está fora do ar. Tente novamente.',
  servico_indisponivel: 'Servidor indisponível. Tente em instantes.',
  catalogo_aquecendo: 'O catálogo ainda está carregando a Pokédex. Tente em alguns segundos.',
  erro_interno: 'Erro interno. Tente novamente.',
  timeout: 'O servidor demorou demais para responder. Tente novamente.',
  rede: 'Falha na conexão com o servidor. Os serviços estão rodando?',
};

export function mostrarErro(err) {
  const msg = MENSAGENS[err?.codigo] ?? err?.message ?? 'Algo deu errado.';
  DialogBox.mostrar(`⚠ ${msg}`, 'erro');
}
