import { API } from '../api.js';
import { Estado } from '../state.js';
import { navegar } from '../router.js';
import { html, escapar, espera, TIER_PT, TIER_ESTRELAS } from '../utils.js';
import { mostrarErro } from '../components/dialog-box.js';
import { criarMaquinaGacha } from '../components/gacha-machine.js';
import { detalhesPokemon } from '../components/pokemon-card.js';
import { barraJogador, PRECO_PACOTE } from './menu.js';

export async function renderGacha(app) {
  const barra = barraJogador();
  const el = html(`
    <section class="gacha">
      <div class="gacha-palco">
        <div class="gacha-fase fase-maquina"></div>
        <div class="gacha-fase fase-revelacao" hidden></div>
      </div>
      <div class="gacha-controles">
        <button type="button" class="btn btn-primario btn-girar">Girar! <small>(${PRECO_PACOTE})</small></button>
        <button type="button" class="btn btn-sec btn-voltar">Voltar</button>
      </div>
      <p class="gacha-chances" aria-live="polite"></p>
    </section>`);
  el.prepend(barra.el);
  app.appendChild(el);

  const maquina = criarMaquinaGacha();
  el.querySelector('.fase-maquina').appendChild(maquina);
  const revelacao = el.querySelector('.fase-revelacao');
  const btnGirar = el.querySelector('.btn-girar');
  const btnVoltar = el.querySelector('.btn-voltar');
  let ocupado = false;

  const atualizarBotao = () => {
    const pode = (Estado.jogador?.moedas ?? 0) >= PRECO_PACOTE;
    btnGirar.disabled = ocupado || !pode;
    btnGirar.title = pode ? '' : 'Moedas insuficientes';
  };
  atualizarBotao();
  const parar = Estado.ouvir('jogador', atualizarBotao);

  API.chancesGacha().then((c) => {
    const txt = c.chances.map((x) => `${TIER_PT[x.tier]} ${x.porcentagem}%`).join(' · ');
    el.querySelector('.gacha-chances').textContent = `${txt} — garantia: épico+ a cada ${c.pity_epico_ou_melhor}, lendário a cada ${c.pity_lendario}`;
  }).catch(() => {});

  btnVoltar.addEventListener('click', () => navegar('menu'));

  btnGirar.addEventListener('click', async () => {
    if (ocupado) return;
    ocupado = true;
    atualizarBotao();
    btnVoltar.disabled = true;

    // Fase 1 -> 2: volta para a máquina e começa a animação junto com o pedido.
    revelacao.hidden = true;
    el.querySelector('.fase-maquina').hidden = false;
    maquina.classList.remove('saindo', 'abrindo');
    maquina.classList.add('girando');
    const animacao = espera(1100);

    try {
      const res = await API.abrirPacote(Estado.jogador.id);
      await animacao;
      maquina.classList.remove('girando');
      maquina.classList.add('saindo');      // pokébola desce
      maquina.style.setProperty('--cor-tier', `var(--tier-${res.pokemon.tier})`);
      await espera(1300);
      maquina.classList.add('abrindo');     // flash
      await espera(450);

      Estado.atualizarMoedas(res.saldo, res.pity);
      Estado.invalidarColecao();
      mostrarRevelacao(res);
    } catch (err) {
      maquina.classList.remove('girando', 'saindo', 'abrindo');
      mostrarErro(err);
    } finally {
      ocupado = false;
      btnVoltar.disabled = false;
      atualizarBotao();
      btnGirar.innerHTML = `Abrir outro <small>(${PRECO_PACOTE})</small>`;
      btnGirar.focus();
    }
  });

  function mostrarRevelacao(res) {
    const p = res.pokemon;
    const item = { ...p, tier: p.detalhes?.tier ?? p.tier, tipos: p.detalhes?.tipos ?? p.tipos };
    revelacao.innerHTML = '';
    const bloco = html(`
      <div class="revelacao tier-${escapar(item.tier)} anim-entrada">
        <p class="revelacao-tier">${escapar(`${TIER_ESTRELAS[item.tier]} ${TIER_PT[item.tier].toUpperCase()} ${TIER_ESTRELAS[item.tier]}`.trim())}</p>
        <div class="janela revelacao-card"></div>
        ${res.garantido_pelo_pity ? '<p class="revelacao-pity">garantido pelo pity!</p>' : ''}
        <p class="revelacao-saldo"><span class="moeda" aria-hidden="true"></span>${res.saldo} restantes</p>
      </div>`);
    bloco.querySelector('.revelacao-card').appendChild(detalhesPokemon(item, { tamanhoSprite: 128 }));
    revelacao.appendChild(bloco);
    el.querySelector('.fase-maquina').hidden = true;
    revelacao.hidden = false;
  }

  return () => { barra.parar(); parar(); };
}
