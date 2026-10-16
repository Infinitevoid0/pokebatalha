import { API } from '../api.js';
import { Estado } from '../state.js';
import { navegar } from '../router.js';
import { html, escapar, TIERS, TIER_PT, TIPOS_PT } from '../utils.js';
import { mostrarErro } from '../components/dialog-box.js';
import { agruparColecao, cardPokemon, abrirModalPokemon, poderDe } from '../components/pokemon-card.js';

const LOTE_RENDER = 40; // cards por vez; o resto entra ao rolar

// busca a coleção, usando o cache do estado.
export async function carregarColecao() {
  if (!Estado.colecao) Estado.colecao = await API.buscarColecao(Estado.jogador.id);
  return Estado.colecao;
}

// grade em lotes com IntersectionObserver; devolve a função que para o observer.
export function renderizarGrade(grade, itens, opcoesCard = () => ({})) {
  grade.innerHTML = '';
  let pos = 0;
  const sentinela = document.createElement('div');
  sentinela.className = 'grade-sentinela';

  const proximoLote = () => {
    const frag = document.createDocumentFragment();
    for (const item of itens.slice(pos, pos + LOTE_RENDER)) frag.appendChild(cardPokemon(item, opcoesCard(item)));
    pos += LOTE_RENDER;
    grade.appendChild(frag);
    if (pos < itens.length) grade.appendChild(sentinela);
    else sentinela.remove();
  };

  const obs = new IntersectionObserver((entradas) => {
    if (entradas.some((e) => e.isIntersecting)) proximoLote();
  }, { rootMargin: '400px' });
  proximoLote();
  obs.observe(sentinela);
  return () => obs.disconnect();
}

export async function renderColecao(app) {
  const el = html(`
    <section class="colecao">
      <header class="cabecalho-tela">
        <h1 class="janela-titulo">Coleção de ${escapar(Estado.jogador.nome)} <small class="contagem"></small></h1>
        <button type="button" class="btn btn-sec" data-acao="voltar">Voltar</button>
      </header>
      <div class="filtros">
        <label>Tier
          <select class="select" data-filtro="tier" aria-label="Filtrar por tier">
            <option value="">Todos</option>
            ${TIERS.map((t) => `<option value="${t}">${TIER_PT[t]}</option>`).join('')}
          </select>
        </label>
        <label>Tipo
          <select class="select" data-filtro="tipo" aria-label="Filtrar por tipo">
            <option value="">Todos</option>
            ${Object.entries(TIPOS_PT).map(([k, v]) => `<option value="${k}">${v}</option>`).join('')}
          </select>
        </label>
        <label>Ordem
          <select class="select" data-filtro="ordem" aria-label="Ordenar">
            <option value="poder">Poder ↓</option>
            <option value="nome">Nome A-Z</option>
            <option value="recente">Mais recentes</option>
          </select>
        </label>
      </div>
      <p class="aviso" hidden></p>
      <div class="grade-cards" role="list"><p class="carregando">Carregando...</p></div>
    </section>`);
  app.appendChild(el);
  el.querySelector('[data-acao="voltar"]').addEventListener('click', () => navegar('menu'));

  let pararGrade = () => {};
  let colecao;
  try {
    colecao = await carregarColecao();
  } catch (err) {
    mostrarErro(err);
    el.querySelector('.grade-cards').innerHTML = '<p class="vazio">Não foi possível carregar a coleção.</p>';
    return;
  }

  const todos = agruparColecao(colecao.pokemons);
  const ordemRecente = new Map(colecao.pokemons.map((p, i) => [p.pokemon_id, -new Date(p.adquirido_em).getTime() || i]));
  el.querySelector('.contagem').textContent = `(${colecao.pokemons.length})`;
  if (colecao.detalhes_indisponiveis) {
    const aviso = el.querySelector('.aviso');
    aviso.hidden = false;
    aviso.textContent = '(catálogo indisponível: mostrando sem stats e poder)';
  }

  const grade = el.querySelector('.grade-cards');
  const filtros = el.querySelector('.filtros');

  function aplicar() {
    const tier = filtros.querySelector('[data-filtro="tier"]').value;
    const tipo = filtros.querySelector('[data-filtro="tipo"]').value;
    const ordem = filtros.querySelector('[data-filtro="ordem"]').value;
    let itens = todos.filter((p) => (!tier || p.tier === tier) && (!tipo || p.tipos?.includes(tipo)));
    if (ordem === 'nome') itens = [...itens].sort((a, b) => a.nome.localeCompare(b.nome));
    else if (ordem === 'recente') itens = [...itens].sort((a, b) => ordemRecente.get(a.pokemon_id) - ordemRecente.get(b.pokemon_id));
    else itens = [...itens].sort((a, b) => (poderDe(b) ?? 0) - (poderDe(a) ?? 0));

    pararGrade();
    if (todos.length === 0) {
      grade.innerHTML = '<p class="vazio">Sua coleção está vazia. Abra um pacote!</p>';
      return;
    }
    if (itens.length === 0) {
      grade.innerHTML = '<p class="vazio">Nenhum Pokémon com esses filtros.</p>';
      return;
    }
    pararGrade = renderizarGrade(grade, itens);
  }
  filtros.addEventListener('change', aplicar);
  aplicar();

  // um listener para a grade toda (delegação).
  grade.addEventListener('click', (e) => {
    const card = e.target.closest('.poke-card');
    if (!card) return;
    const item = todos.find((p) => p.pokemon_id === Number(card.dataset.id));
    if (item) abrirModalPokemon(item);
  });

  return () => pararGrade();
}
