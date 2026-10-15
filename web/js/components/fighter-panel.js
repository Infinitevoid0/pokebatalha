// Painel de um lado da batalha: sprite na arena + caixa de status
// (nome, nível, tier, tipos, habilidade, HP e estágios).
import { exibir, escapar, sinal } from '../utils.js';
import { imgPokemon } from '../sprites.js';
import { tiposHTML } from './tipo-badge.js';
import { tierBadge } from './tier-badge.js';
import { criarHPBar, atualizarHPBar } from './hp-bar.js';
import { criarTeamBar } from './team-bar.js';

export function criarFighterPanel(lado) {
  const el = document.createElement('div');
  el.className = `fighter lado-${lado}`;
  el.innerHTML = `
    <div class="fighter-status janela"></div>
    <div class="fighter-palco">
      <div class="plataforma" aria-hidden="true"></div>
      <div class="fighter-sprite"></div>
    </div>`;
  return el;
}

// Redesenha o painel com o lutador ativo do lado.
export function desenharFighter(el, ladoBatalha, { comTime = true } = {}) {
  const l = ladoBatalha.time[ladoBatalha.ativo];
  const rival = el.classList.contains('lado-rival');
  el.dataset.nome = exibir(l.nome);
  el.classList.toggle('caido', l.hp <= 0);

  const estagios = [];
  if (l.estagio_ataque) estagios.push(`ATQ ${sinal(l.estagio_ataque)}`);
  if (l.estagio_velocidade) estagios.push(`VEL ${sinal(l.estagio_velocidade)}`);

  const status = el.querySelector('.fighter-status');
  status.innerHTML = `
    <div class="fs-linha1">
      <span class="fs-nome">${escapar(exibir(l.nome))}</span>
      <span class="fs-nivel">Nv50</span>
    </div>
    <div class="fs-linha2">${tierBadge(l.tier)} <span class="fs-poder">PODER ${l.poder}</span></div>
    <div class="fs-linha2">${tiposHTML(l.tipos)}</div>
    <p class="fs-hab" title="${escapar(l.descricao_habilidade || '')}">${l.habilidade ? escapar(exibir(l.habilidade)) : 'Sem habilidade'}</p>
    <div class="fs-hp"></div>
    <p class="fs-estagios">${estagios.join(' · ') || '&nbsp;'}</p>`;
  const hp = criarHPBar(l.hp, l.hp_max);
  status.querySelector('.fs-hp').appendChild(hp);
  if (comTime) status.appendChild(criarTeamBar(ladoBatalha, { detalhado: rival }));

  const spriteBox = el.querySelector('.fighter-sprite');
  spriteBox.innerHTML = '';
  spriteBox.appendChild(imgPokemon({
    id: l.pokemon_id, nome: exibir(l.nome), tipo: rival ? 'frente' : 'costas',
    tamanho: 128, preguicoso: false, classe: 'sprite-batalha',
  }));
}

// Atualiza só o HP (com animação), sem redesenhar o resto.
export function atualizarHPFighter(el, hp, max) {
  const barra = el.querySelector('.hp');
  if (barra) atualizarHPBar(barra, hp, max, true);
  el.classList.toggle('caido', hp <= 0);
}
