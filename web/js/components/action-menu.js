// menu de ações: só o que está em acoes_disponiveis; "trocar" abre um submenu.
import { exibir, escapar, traduzirTipo } from '../utils.js';
import { imgPokemon } from '../sprites.js';

function rotulo(acao, lutador) {
  const tipo = (i) => lutador.tipos?.[i] ?? 'normal';
  switch (acao) {
    case 'fisico': return { titulo: 'Ataque Físico', sub: 'poder 50 · neutro', cor: 'var(--type-normal)' };
    case 'tipo': return { titulo: `Ataque de ${traduzirTipo(tipo(0))}`, sub: 'poder 70', cor: `var(--type-${tipo(0)})` };
    case 'tipo2': return { titulo: `Ataque de ${traduzirTipo(tipo(1))}`, sub: 'poder 70', cor: `var(--type-${tipo(1)})` };
    case 'fake_out': return { titulo: 'Fake Out', sub: 'prioridade · faz recuar', cor: 'var(--accent-yellow)' };
    case 'protect': return { titulo: 'Protect', sub: 'bloqueia 1 golpe', cor: '#8fd3ff' };
    case 'trocar': return { titulo: 'Trocar', sub: 'vai primeiro', cor: 'var(--accent-green)' };
    default: return { titulo: acao, sub: '', cor: 'var(--bg-surface)' };
  }
}

// onEscolher(acao, alvo) roda quando o jogador decide.
export function criarActionMenu(onEscolher) {
  const el = document.createElement('div');
  el.className = 'action-menu janela';
  let teclas = [];

  function botaoAcao(acao, lutador, n) {
    const r = rotulo(acao, lutador);
    const b = document.createElement('button');
    b.type = 'button';
    b.className = 'btn acao';
    b.style.setProperty('--cor-acao', r.cor);
    b.dataset.acao = acao;
    b.innerHTML = `<span class="acao-num">${n}</span><span class="acao-titulo">${escapar(r.titulo)}</span><span class="acao-sub">${escapar(r.sub)}</span>`;
    b.setAttribute('aria-label', `${r.titulo}, ${r.sub}`);
    return b;
  }

  function botaoTroca(lutador, idx, n) {
    const b = document.createElement('button');
    b.type = 'button';
    b.className = 'btn acao troca';
    b.dataset.alvo = String(idx);
    b.innerHTML = `<span class="acao-num">${n}</span>`;
    b.appendChild(imgPokemon({ id: lutador.pokemon_id, nome: exibir(lutador.nome), tamanho: 40, preguicoso: false }));
    b.insertAdjacentHTML('beforeend', `<span class="acao-titulo">${escapar(exibir(lutador.nome))}</span><span class="acao-sub">HP ${lutador.hp}/${lutador.hp_max}</span>`);
    b.setAttribute('aria-label', `Trocar para ${exibir(lutador.nome)}, HP ${lutador.hp} de ${lutador.hp_max}`);
    return b;
  }

  function mostrarTrocas(batalha, forcada) {
    const time = batalha.jogador.time;
    el.innerHTML = `<p class="menu-titulo">${forcada ? 'Seu Pokémon desmaiou! Escolha quem entra.' : 'Trocar por quem?'}</p><div class="acoes"></div>`;
    const grade = el.querySelector('.acoes');
    teclas = [];
    batalha.trocas_disponiveis.forEach((idx, i) => {
      const b = botaoTroca(time[idx], idx, i + 1);
      b.addEventListener('click', () => onEscolher('trocar', idx));
      grade.appendChild(b);
      teclas.push(b);
    });
    if (!forcada) {
      const voltar = document.createElement('button');
      voltar.type = 'button';
      voltar.className = 'btn btn-sec';
      voltar.textContent = 'Voltar';
      voltar.addEventListener('click', () => mostrar(batalha));
      el.appendChild(voltar);
    }
    teclas[0]?.focus();
  }

  function mostrar(batalha) {
    if (batalha.precisa_trocar) { mostrarTrocas(batalha, true); return; }
    const ativo = batalha.jogador.time[batalha.jogador.ativo];
    el.innerHTML = `<p class="menu-titulo">O que ${escapar(exibir(ativo.nome))} vai fazer?</p><div class="acoes"></div>`;
    const grade = el.querySelector('.acoes');
    teclas = [];
    batalha.acoes_disponiveis.forEach((acao, i) => {
      const b = botaoAcao(acao, ativo, i + 1);
      b.addEventListener('click', () => {
        if (acao === 'trocar') mostrarTrocas(batalha, false);
        else onEscolher(acao, 0);
      });
      grade.appendChild(b);
      teclas.push(b);
    });
    teclas[0]?.focus();
  }

  function aoTeclar(e) {
    if (el.classList.contains('bloqueado') || e.target.tagName === 'INPUT') return;
    const n = Number(e.key);
    if (n >= 1 && n <= teclas.length) { e.preventDefault(); teclas[n - 1].click(); }
  }
  document.addEventListener('keydown', aoTeclar);

  return {
    el,
    mostrar,
    bloquear(sim, texto = 'Aguardando...') {
      el.classList.toggle('bloqueado', sim);
      el.querySelectorAll('button').forEach((b) => { b.disabled = sim; });
      if (sim) el.innerHTML = `<p class="menu-titulo"><span class="carregando">${escapar(texto)}</span></p>`;
    },
    destruir() { document.removeEventListener('keydown', aoTeclar); },
  };
}
