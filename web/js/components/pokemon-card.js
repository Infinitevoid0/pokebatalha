// card de Pokémon (coleção, time, gacha) e modal de detalhes.
import { exibir, escapar, sinal, html } from '../utils.js';
import { imgPokemon } from '../sprites.js';
import { tiposHTML } from './tipo-badge.js';
import { tierBadge } from './tier-badge.js';

// agrupa repetidos e conta; mantém a ordem da API (mais forte primeiro).
export function agruparColecao(pokemons = []) {
  const grupos = new Map();
  for (const p of pokemons) {
    const g = grupos.get(p.pokemon_id);
    if (g) g.qtd++;
    else grupos.set(p.pokemon_id, { ...p, tier: p.detalhes?.tier ?? p.tier, tipos: p.detalhes?.tipos ?? p.tipos, qtd: 1 });
  }
  return [...grupos.values()];
}

export function poderDe(item) { return item.detalhes?.poder?.total ?? null; }

export function cardPokemon(item, { ordem = 0, selecionado = false } = {}) {
  const poder = poderDe(item);
  const el = html(`
    <button type="button" class="poke-card tier-${escapar(item.tier)}${selecionado ? ' selecionado' : ''}"
            data-id="${item.pokemon_id}"
            aria-label="${escapar(exibir(item.nome))}, ${escapar(item.tier)}${poder != null ? `, poder ${poder}` : ''}"
            aria-pressed="${selecionado}">
      ${ordem ? `<span class="ordem" aria-hidden="true">${ordem}</span>` : ''}
      ${item.qtd > 1 ? `<span class="qtd">x${item.qtd}</span>` : ''}
      <span class="nome">${escapar(exibir(item.nome))}</span>
      ${tierBadge(item.tier)}
      <span class="poder">${poder ?? '?'} <small>PODER</small></span>
      ${tiposHTML(item.tipos)}
    </button>`);
  el.insertBefore(imgPokemon({ id: item.pokemon_id, nome: exibir(item.nome), tipo: 'frente', tamanho: 80 }), el.querySelector('.nome'));
  return el;
}

const NOMES_STATS = [
  ['hp', 'HP'], ['ataque', 'ATQ'], ['defesa', 'DEF'],
  ['ataque_especial', 'AT.E'], ['defesa_especial', 'DF.E'], ['velocidade', 'VEL'],
];

export function statsHTML(stats) {
  if (!stats) return '<p class="composicao">Stats indisponíveis (catálogo fora do ar).</p>';
  return `<div class="stats">${NOMES_STATS.map(([k, rotulo]) => `
    <span>${rotulo}</span>
    <span class="barra"><span style="transform: scaleX(${Math.min(stats[k] / 180, 1)})"></span></span>
    <span>${stats[k]}</span>`).join('')}</div>`;
}

export function composicaoHTML(poder) {
  if (!poder) return '';
  const partes = [`stats <b>${poder.stats}</b>`, `tipagem <b>${sinal(poder.tipagem)}</b>`];
  if (poder.habilidade > 0) partes.push(`${escapar(exibir(poder.habilidade_ativa))} <b>${sinal(poder.habilidade)}</b>`);
  if (poder.fake_out > 0) partes.push(`Fake Out <b>${sinal(poder.fake_out)}</b>`);
  return `<p class="composicao">PODER <b>${poder.total}</b> = ${partes.join(' + ')}</p>`;
}

// sprite grande, badges, habilidade, poder e stats.
export function detalhesPokemon(item, { tamanhoSprite = 96 } = {}) {
  const d = item.detalhes;
  const el = html(`
    <div class="detalhes-poke">
      <div class="detalhes-topo">
        <div class="detalhes-sprite"></div>
        <div class="detalhes-info">
          <h3 class="detalhes-nome">${escapar(exibir(item.nome))}</h3>
          ${tierBadge(item.tier)}
          ${tiposHTML(item.tipos)}
          <p class="detalhes-hab">${d?.poder?.habilidade_ativa ? `Habilidade: ${escapar(exibir(d.poder.habilidade_ativa))}` : 'Sem habilidade especial'}</p>
        </div>
      </div>
      ${composicaoHTML(d?.poder)}
      ${statsHTML(d?.stats)}
    </div>`);
  el.querySelector('.detalhes-sprite').appendChild(
    imgPokemon({ id: item.pokemon_id, nome: exibir(item.nome), tipo: 'frente', tamanho: tamanhoSprite, preguicoso: false }));
  return el;
}

export function abrirModalPokemon(item) {
  const anterior = document.activeElement;
  const fundo = html(`
    <div class="modal-fundo" role="dialog" aria-modal="true" aria-label="Detalhes de ${escapar(exibir(item.nome))}">
      <div class="modal janela anim-entrada">
        <div class="modal-corpo"></div>
        <button type="button" class="btn btn-sec modal-fechar">Fechar</button>
      </div>
    </div>`);
  fundo.querySelector('.modal-corpo').appendChild(detalhesPokemon(item));
  const fechar = () => {
    fundo.remove();
    document.removeEventListener('keydown', aoTeclar);
    anterior?.focus?.();
  };
  const aoTeclar = (e) => { if (e.key === 'Escape') fechar(); };
  fundo.addEventListener('click', (e) => { if (e.target === fundo) fechar(); });
  fundo.querySelector('.modal-fechar').addEventListener('click', fechar);
  document.addEventListener('keydown', aoTeclar);
  document.body.appendChild(fundo);
  fundo.querySelector('.modal-fechar').focus();
}
