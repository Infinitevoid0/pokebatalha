import { API } from '../api.js';
import { Estado } from '../state.js';
import { navegar } from '../router.js';
import { html, escapar, exibir, animar } from '../utils.js';
import { mostrarErro, DialogBox } from '../components/dialog-box.js';
import { agruparColecao } from '../components/pokemon-card.js';
import { tiposHTML } from '../components/tipo-badge.js';
import { criarFighterPanel, desenharFighter, atualizarHPFighter } from '../components/fighter-panel.js';
import { criarLogPanel } from '../components/log-panel.js';
import { criarActionMenu } from '../components/action-menu.js';
import { precarregarSprites } from '../sprites.js';
import { carregarColecao, renderizarGrade } from './colecao.js';

const TAMANHO_TIME = 3;
const CLIMA_PT = { sol: 'Sol forte', chuva: 'Chuva' };

/* ===================== Seleção de time ===================== */

export async function renderSelecaoTime(app) {
  const el = html(`
    <section class="selecao">
      <header class="cabecalho-tela">
        <h1 class="janela-titulo">Montar time <small>(máx. ${TAMANHO_TIME})</small></h1>
        <button type="button" class="btn btn-sec" data-acao="voltar">Voltar</button>
      </header>
      <div class="selecao-corpo">
        <div class="selecao-colecao">
          <p class="rotulo">Sua coleção (clique para escolher):</p>
          <div class="grade-cards"><p class="carregando">Carregando...</p></div>
        </div>
        <aside class="selecao-time janela">
          <p class="rotulo">Time selecionado</p>
          <ol class="lista-time"></ol>
          <p class="dica">O primeiro começa em campo. O rival recebe Pokémon do mesmo tier e poder parecido.</p>
          <button type="button" class="btn btn-primario btn-iniciar" disabled>Iniciar batalha</button>
        </aside>
      </div>
    </section>`);
  app.appendChild(el);
  el.querySelector('[data-acao="voltar"]').addEventListener('click', () => navegar('menu'));

  let colecao;
  try {
    colecao = await carregarColecao();
  } catch (err) {
    mostrarErro(err);
    el.querySelector('.grade-cards').innerHTML = '<p class="vazio">Não foi possível carregar a coleção.</p>';
    return;
  }
  const itens = agruparColecao(colecao.pokemons);
  const porId = new Map(itens.map((p) => [p.pokemon_id, p]));
  const grade = el.querySelector('.grade-cards');
  const lista = el.querySelector('.lista-time');
  const btnIniciar = el.querySelector('.btn-iniciar');

  if (itens.length === 0) {
    grade.innerHTML = '<p class="vazio">Você ainda não tem Pokémon. Abra um pacote primeiro!</p>';
    return;
  }

  // Mantém o último time usado, se ainda estiver na coleção.
  let time = Estado.time.filter((id) => porId.has(id)).slice(0, TAMANHO_TIME);
  let pararGrade = () => {};

  function desenhar() {
    pararGrade();
    pararGrade = renderizarGrade(grade, itens, (item) => {
      const pos = time.indexOf(item.pokemon_id);
      return { ordem: pos + 1, selecionado: pos >= 0 };
    });
    lista.innerHTML = '';
    for (let i = 0; i < TAMANHO_TIME; i++) {
      const p = porId.get(time[i]);
      lista.insertAdjacentHTML('beforeend', p
        ? `<li><span class="lt-num">${i + 1}.</span> <span class="lt-nome">${escapar(exibir(p.nome))}</span> ${tiposHTML(p.tipos)} <span class="lt-poder">${p.detalhes?.poder?.total ?? '?'}</span></li>`
        : `<li class="vazio-slot"><span class="lt-num">${i + 1}.</span> (vazio)</li>`);
    }
    btnIniciar.disabled = time.length === 0;
  }
  desenhar();

  grade.addEventListener('click', (e) => {
    const card = e.target.closest('.poke-card');
    if (!card) return;
    const id = Number(card.dataset.id);
    if (time.includes(id)) time = time.filter((x) => x !== id);
    else if (time.length < TAMANHO_TIME) time = [...time, id];
    else { DialogBox.mostrar(`O time tem no máximo ${TAMANHO_TIME} Pokémon.`, 'info', 2500); return; }
    const rolagem = window.scrollY;
    desenhar();
    window.scrollTo(0, rolagem);
  });

  btnIniciar.addEventListener('click', async () => {
    btnIniciar.disabled = true;
    btnIniciar.innerHTML = '<span class="carregando">Montando rival...</span>';
    try {
      const b = await API.criarBatalha(Estado.jogador.id, time);
      Estado.time = time;
      Estado.batalha = b;
      navegar('batalha');
    } catch (err) {
      mostrarErro(err);
      btnIniciar.textContent = 'Iniciar batalha';
      btnIniciar.disabled = time.length === 0;
    }
  });

  return () => pararGrade();
}

/* ===================== Batalha ===================== */

// Lê a linha do log e diz de que lado ela fala ("Seu X" ou "X do rival").
function ladoDaLinha(linha) {
  if (/^Seu /.test(linha)) return 'jogador';
  if (/ do rival\b/.test(linha)) return 'rival';
  return null;
}

const RE_DANO = /perdeu (\d+) de HP \((\d+)\/(\d+)\)/;
const RE_TROCA_RIVAL = /O rival (?:envia|recolhe .+? e envia) (.+)!$/;
const RE_TROCA_JOGADOR = /Vai, (.+)!$/;

export async function renderBatalha(app) {
  if (!Estado.batalha) { navegar('time'); return; }

  const el = html(`
    <section class="batalha">
      <header class="batalha-topo">
        <span class="bt-turno"></span>
        <span class="bt-clima"></span>
        <button type="button" class="btn btn-sec btn-desistir">Sair</button>
      </header>
      <div class="arena"></div>
      <div class="batalha-baixo"></div>
    </section>`);
  app.appendChild(el);

  const painelRival = criarFighterPanel('rival');
  const painelJogador = criarFighterPanel('jogador');
  el.querySelector('.arena').append(painelRival, painelJogador);
  const paineis = { rival: painelRival, jogador: painelJogador };

  const log = criarLogPanel();
  const menu = criarActionMenu(jogar);
  el.querySelector('.batalha-baixo').append(log.el, menu.el);

  let atual = Estado.batalha;
  let enviando = false;
  precarregarSprites(atual.cpu.time.map((l) => l.pokemon_id), 'frente');
  precarregarSprites(atual.jogador.time.map((l) => l.pokemon_id), 'costas');

  el.querySelector('.btn-desistir').addEventListener('click', () => {
    if (atual.status !== 'em_andamento' || confirm('Sair da batalha? Ela fica sem resultado.')) navegar('menu');
  });

  function desenharTopo(b) {
    el.querySelector('.bt-turno').textContent = `Turno ${b.turno + (b.status === 'em_andamento' ? 1 : 0)}`;
    el.querySelector('.bt-clima').textContent = b.clima ? `${CLIMA_PT[b.clima] ?? b.clima} (${b.turnos_clima})` : '';
  }

  function desenharTudo(b) {
    desenharFighter(painelJogador, b.jogador);
    desenharFighter(painelRival, b.cpu);
    desenharTopo(b);
  }

  // Reproduz o log do turno linha a linha, aplicando cada efeito no
  // estado visual (que começa igual ao estado anterior).
  async function reproduzir(anterior, novo) {
    const visual = structuredClone(anterior);
    const ladoVisual = (lado) => (lado === 'rival' ? visual.cpu : visual.jogador);
    let efeitoPendente = null;
    log.reiniciarVelocidade();
    log.seta(false);

    for (const linha of novo.log) {
      const lado = ladoDaLinha(linha);
      const painel = lado ? paineis[lado] : null;
      let classe = '';

      if (/ usou /.test(linha) && painel) {
        animar(painel, 'fx-ataca', 300);
      } else if (/É super efetivo/.test(linha)) {
        efeitoPendente = 'fx-super'; classe = 'log-super';
      } else if (/Não é muito efetivo/.test(linha)) {
        efeitoPendente = 'fx-fraco'; classe = 'log-fraco';
      } else if (RE_DANO.test(linha) && painel) {
        const [, , hp, max] = linha.match(RE_DANO).map(Number);
        const l = ladoVisual(lado);
        l.time[l.ativo].hp = hp;
        atualizarHPFighter(painel, hp, max);
        animar(painel.querySelector('.fighter-sprite'), 'fx-dano', 450);
        if (efeitoPendente) animar(painel, efeitoPendente, 1000);
        efeitoPendente = null;
      } else if (/desmaiou!/.test(linha) && painel) {
        painel.classList.add('fx-desmaio');
        classe = 'log-desmaio';
      } else if (/(está se protegendo|se protegeu do golpe)/.test(linha) && painel) {
        painel.querySelector('.fighter-palco').insertAdjacentHTML('beforeend', '<div class="escudo" aria-hidden="true"></div>');
        setTimeout(() => painel.querySelector('.escudo')?.remove(), 1000);
      } else if (RE_TROCA_RIVAL.test(linha) || RE_TROCA_JOGADOR.test(linha)) {
        const rival = RE_TROCA_RIVAL.test(linha);
        const nome = (rival ? linha.match(RE_TROCA_RIVAL) : linha.match(RE_TROCA_JOGADOR))[1];
        const l = rival ? visual.cpu : visual.jogador;
        const idx = l.time.findIndex((p, i) => exibir(p.nome) === nome && i !== l.ativo && p.hp > 0);
        if (idx >= 0) {
          l.ativo = idx;
          const p = rival ? painelRival : painelJogador;
          p.classList.remove('fx-desmaio');
          desenharFighter(p, l);
          animar(p, 'fx-entrada', 450);
        }
      } else if (/Você venceu!/.test(linha)) classe = 'log-vitoria';
      else if (/Você perdeu!/.test(linha)) classe = 'log-derrota';

      await log.escrever(linha, classe);
    }
  }

  async function aplicarNovoEstado(anterior, novo) {
    menu.bloquear(true, '...');
    await reproduzir(anterior, novo);
    paineis.jogador.classList.remove('fx-desmaio');
    paineis.rival.classList.remove('fx-desmaio');
    atual = novo;
    Estado.batalha = novo;
    desenharTudo(novo);
    if (novo.status !== 'em_andamento') {
      mostrarResultado(novo);
      return;
    }
    menu.bloquear(false);
    menu.mostrar(novo);
  }

  async function jogar(acao, alvo) {
    if (enviando) return; // evita clique duplo
    enviando = true;
    menu.bloquear(true);
    try {
      const novo = await API.jogarTurno(atual.id, acao, alvo);
      log.limpar();
      await aplicarNovoEstado(atual, novo);
    } catch (err) {
      if (err.codigo === 'batalha_encerrada') { navegar('menu'); return; }
      mostrarErro(err);
      // Em conflito (ex: duas abas), recarrega o estado real da batalha.
      if (err.codigo === 'conflito' || err.codigo === 'batalha_nao_encontrada') {
        try { atual = await API.buscarBatalha(atual.id); desenharTudo(atual); } catch { /* segue com o que tem */ }
      }
      menu.bloquear(false);
      menu.mostrar(atual);
    } finally {
      enviando = false;
    }
  }

  function mostrarResultado(b) {
    const vitoria = b.status === 'vitoria';
    if (vitoria) {
      Estado.atualizarMoedas((Estado.jogador.moedas ?? 0) + (b.recompensa || 0));
      soltarConfete();
    } else {
      document.body.insertAdjacentHTML('beforeend', '<div class="tela-escura"></div>');
    }
    const fundo = html(`
      <div class="modal-fundo resultado" role="dialog" aria-modal="true">
        <div class="modal janela anim-entrada resultado-caixa ${vitoria ? 'vitoria' : 'derrota'}">
          <h2 class="janela-titulo">${vitoria ? 'VITÓRIA!' : 'Derrota...'}</h2>
          <p>${vitoria ? `+${b.recompensa} moedas` : 'Abra mais pacotes e tente de novo.'}</p>
          <p class="resultado-turnos">${b.turno} turnos</p>
          <div class="resultado-botoes">
            <button type="button" class="btn btn-primario" data-ir="time">Batalhar de novo</button>
            <button type="button" class="btn" data-ir="menu">Menu</button>
          </div>
        </div>
      </div>`);
    fundo.addEventListener('click', (e) => {
      const destino = e.target.closest('[data-ir]')?.dataset.ir;
      if (!destino) return;
      Estado.batalha = null;
      navegar(destino);
    });
    document.body.appendChild(fundo);
    fundo.querySelector('[data-ir]').focus();
  }

  // Primeiro desenho: estado inicial e o log de abertura (habilidades de entrada).
  desenharTudo(atual);
  const inicio = structuredClone(atual);
  log.limpar();
  aplicarNovoEstado(inicio, atual);

  return () => {
    menu.destruir();
    document.querySelectorAll('.resultado, .tela-escura, .confete').forEach((n) => n.remove());
  };
}

function soltarConfete() {
  const cores = ['#f7d51d', '#e43b44', '#0095e9', '#3e8948', '#b55088', '#f2f2f2'];
  const frag = document.createDocumentFragment();
  for (let i = 0; i < 60; i++) {
    const c = document.createElement('span');
    c.className = 'confete';
    c.style.left = `${Math.random() * 100}vw`;
    c.style.background = cores[i % cores.length];
    c.style.animationDuration = `${1.8 + Math.random() * 1.6}s`;
    c.style.animationDelay = `${Math.random() * 0.6}s`;
    frag.appendChild(c);
  }
  document.body.appendChild(frag);
  setTimeout(() => document.querySelectorAll('.confete').forEach((n) => n.remove()), 4500);
}
