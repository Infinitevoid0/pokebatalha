import { API } from '../api.js';
import { Estado } from '../state.js';
import { navegar } from '../router.js';
import { html, escapar, animar } from '../utils.js';
import { DialogBox, mostrarErro } from '../components/dialog-box.js';

function bolasDeFundo() {
  let s = '';
  for (let i = 0; i < 9; i++) {
    const x = (i * 37 + 7) % 100;
    const y = (i * 53 + 11) % 100;
    const t = 4 + (i % 4) * 1.5;
    s += `<img class="bola-fundo" src="assets/img/pokeball.svg" alt="" style="left:${x}%;top:${y}%;animation-duration:${t}s;animation-delay:-${i}s;width:${24 + (i % 3) * 16}px">`;
  }
  return `<div class="fundo-bolas" aria-hidden="true">${s}</div>`;
}

export async function renderLogin(app) {
  const idSalvo = Estado.idSalvo();
  const el = html(`
    <section class="login">
      ${bolasDeFundo()}
      <h1 class="logo">★ POKÉ<span>BATALHA</span> ★</h1>
      <p class="subtitulo">colecione, monte seu time, vença o rival</p>

      <div class="janela login-caixa">
        ${idSalvo ? `<button type="button" class="btn btn-primario btn-continuar" data-acao="continuar">Continuar como #${escapar(idSalvo)}</button>
        <div class="separador"><span>ou</span></div>` : ''}

        <form class="form-entrar" novalidate>
          <div class="campo">
            <label for="campo-id">Seu ID de jogador</label>
            <input id="campo-id" class="input" inputmode="numeric" autocomplete="off" placeholder="ex: 12" aria-label="ID do jogador">
          </div>
          <button type="submit" class="btn">Entrar</button>
        </form>

        <div class="separador"><span>ou</span></div>

        <form class="form-criar" novalidate>
          <div class="campo">
            <label for="campo-nome">Nome para nova conta</label>
            <input id="campo-nome" class="input" maxlength="20" autocomplete="off" placeholder="3 a 20 letras" aria-label="Nome da nova conta">
          </div>
          <button type="submit" class="btn btn-primario">Criar conta</button>
        </form>
      </div>
    </section>`);
  app.appendChild(el);

  const caixa = el.querySelector('.login-caixa');
  const ocupado = (sim) => caixa.querySelectorAll('button, input').forEach((b) => { b.disabled = sim; });

  async function entrar(id) {
    ocupado(true);
    try {
      const j = await API.buscarJogador(id);
      Estado.setJogador(j);
      navegar('menu');
    } catch (err) {
      animar(caixa, 'anim-shake', 400);
      mostrarErro(err);
      ocupado(false);
    }
  }

  el.querySelector('[data-acao="continuar"]')?.addEventListener('click', () => entrar(idSalvo));

  el.querySelector('.form-entrar').addEventListener('submit', (e) => {
    e.preventDefault();
    const id = el.querySelector('#campo-id').value.trim();
    if (!/^\d+$/.test(id)) {
      animar(caixa, 'anim-shake', 400);
      DialogBox.mostrar('⚠ Digite o número do seu ID.', 'erro');
      return;
    }
    entrar(id);
  });

  el.querySelector('.form-criar').addEventListener('submit', async (e) => {
    e.preventDefault();
    const nome = el.querySelector('#campo-nome').value.trim();
    if (nome.length < 3) {
      animar(caixa, 'anim-shake', 400);
      DialogBox.mostrar('⚠ O nome precisa ter de 3 a 20 caracteres.', 'erro');
      return;
    }
    ocupado(true);
    try {
      const j = await API.criarJogador(nome);
      Estado.setJogador(j);
      navegar('menu');
      DialogBox.mostrar(`Conta criada! Seu ID é ${j.id}. Guarde para entrar depois.`, 'info', 7000);
    } catch (err) {
      animar(caixa, 'anim-shake', 400);
      mostrarErro(err);
      ocupado(false);
    }
  });

  (idSalvo ? el.querySelector('[data-acao="continuar"]') : el.querySelector('#campo-id')).focus();
}
