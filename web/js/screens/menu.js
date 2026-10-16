import { API } from '../api.js';
import { Estado } from '../state.js';
import { navegar } from '../router.js';
import { html, escapar } from '../utils.js';
import { mostrarErro } from '../components/dialog-box.js';

export const PRECO_PACOTE = 30;

// barra com nome, moedas e pity (menu e gacha).
export function barraJogador() {
  const el = document.createElement('div');
  el.className = 'barra-jogador janela';
  const desenhar = (j) => {
    if (!j) return;
    el.innerHTML = `
      <span class="bj-nome">${escapar(j.nome)} <small>#${j.id}</small></span>
      <span class="bj-moedas" title="Moedas"><span class="moeda" aria-hidden="true"></span>${j.moedas}</span>
      <span class="bj-pity" title="Pacotes até a garantia">
        <span>ÉPICO+ em <b>${j.pity?.epico_garantido_em ?? '?'}</b></span>
        <span>LENDÁRIO em <b>${j.pity?.lendario_garantido_em ?? '?'}</b></span>
      </span>`;
  };
  desenhar(Estado.jogador);
  const parar = Estado.ouvir('jogador', desenhar);
  return { el, parar };
}

export async function renderMenu(app) {
  const barra = barraJogador();
  const el = html(`
    <section class="menu">
      <h1 class="janela-titulo menu-titulo">Olá, ${escapar(Estado.jogador.nome)}!</h1>
      <div class="menu-grade">
        <button type="button" class="btn menu-opcao opcao-gacha" data-ir="gacha">
          <span class="icone-pixel icone-pacote" aria-hidden="true"></span>
          Abrir pacote <small>${PRECO_PACOTE} moedas</small>
        </button>
        <button type="button" class="btn menu-opcao" data-ir="colecao">
          <span class="icone-pixel icone-colecao" aria-hidden="true"></span>
          Ver coleção
        </button>
        <button type="button" class="btn menu-opcao opcao-batalha" data-ir="time">
          <span class="icone-pixel icone-espadas" aria-hidden="true"></span>
          Batalhar
        </button>
        <button type="button" class="btn menu-opcao opcao-sair" data-ir="sair">
          <span class="icone-pixel icone-porta" aria-hidden="true"></span>
          Sair
        </button>
      </div>
      <p class="dica-menu"></p>
    </section>`);
  el.prepend(barra.el);
  app.appendChild(el);

  const botaoGacha = el.querySelector('.opcao-gacha');
  const dica = el.querySelector('.dica-menu');
  const atualizarBotoes = (j) => {
    const semMoedas = (j?.moedas ?? 0) < PRECO_PACOTE;
    botaoGacha.disabled = semMoedas;
    botaoGacha.title = semMoedas ? 'Moedas insuficientes: vença batalhas para ganhar mais' : '';
    dica.textContent = semMoedas ? 'Sem moedas para pacotes: vença batalhas para ganhar +50.' : '';
  };
  atualizarBotoes(Estado.jogador);
  const parar = Estado.ouvir('jogador', atualizarBotoes);

  el.querySelector('.menu-grade').addEventListener('click', (e) => {
    const destino = e.target.closest('[data-ir]')?.dataset.ir;
    if (!destino) return;
    if (destino === 'sair') {
      Estado.sair();
      navegar('login');
      return;
    }
    navegar(destino);
  });

  // atualiza as moedas ao voltar ao menu.
  API.buscarJogador(Estado.jogador.id)
    .then((j) => Estado.setJogador(j))
    .catch(mostrarErro);

  el.querySelector('.menu-opcao:not(:disabled)')?.focus();
  return () => { barra.parar(); parar(); };
}
