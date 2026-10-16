// chamadas aos microsserviços; o NGINX encaminha /api/<servico>/ (sem CORS).

const BASE = {
  jogadores: '/api/jogadores',
  batalha: '/api/batalha',
  catalogo: '/api/catalogo',
};

const TIMEOUT_MS = 30000;
const TENTATIVAS_GET = 3;

export class APIError extends Error {
  constructor(status, codigo, mensagem) {
    super(mensagem || `Erro ${status}`);
    this.status = status;
    this.codigo = codigo || '';
  }
}

const espera = (ms) => new Promise((r) => setTimeout(r, ms));

async function chamarUmaVez(metodo, url, corpo) {
  const controle = new AbortController();
  const timer = setTimeout(() => controle.abort(), TIMEOUT_MS);
  const opts = { method: metodo, headers: { Accept: 'application/json' }, signal: controle.signal };
  if (corpo !== undefined) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(corpo);
  }
  try {
    const resp = await fetch(url, opts);
    if (!resp.ok) {
      const erro = await resp.json().catch(() => ({}));
      throw new APIError(resp.status, erro.codigo, erro.mensagem);
    }
    return await resp.json();
  } catch (err) {
    if (err instanceof APIError) throw err;
    if (err.name === 'AbortError') throw new APIError(0, 'timeout', 'O servidor demorou demais para responder.');
    throw new APIError(0, 'rede', 'Não foi possível falar com o servidor.');
  } finally {
    clearTimeout(timer);
  }
}

// GET repete em 502/503/rede com espera crescente; POST nunca (pacote cobra moedas).
async function chamar(metodo, url, corpo) {
  const tentativas = metodo === 'GET' ? TENTATIVAS_GET : 1;
  for (let i = 1; ; i++) {
    try {
      return await chamarUmaVez(metodo, url, corpo);
    } catch (err) {
      const temporario = [0, 502, 503].includes(err.status) && err.codigo !== 'timeout';
      if (!temporario || i >= tentativas) throw err;
      await espera(400 * 2 ** (i - 1));
    }
  }
}

export const API = {
  // jogadores
  criarJogador: (nome) => chamar('POST', `${BASE.jogadores}/jogadores`, { nome }),
  buscarJogador: (id) => chamar('GET', `${BASE.jogadores}/jogadores/${encodeURIComponent(id)}`),
  abrirPacote: (id) => chamar('POST', `${BASE.jogadores}/jogadores/${id}/pacotes`),
  buscarColecao: (id) => chamar('GET', `${BASE.jogadores}/jogadores/${id}/colecao`),
  chancesGacha: () => chamar('GET', `${BASE.jogadores}/gacha/chances`),

  // batalha
  criarBatalha: (jogadorID, time) => chamar('POST', `${BASE.batalha}/batalhas`, { jogador_id: jogadorID, time }),
  buscarBatalha: (id) => chamar('GET', `${BASE.batalha}/batalhas/${id}`),
  jogarTurno: (id, acao, alvo = 0) => chamar('POST', `${BASE.batalha}/batalhas/${id}/turnos`, { acao, alvo }),
  listarHabilidades: () => chamar('GET', `${BASE.batalha}/habilidades`),

  // catálogo
  buscarPokemon: (idOuNome) => chamar('GET', `${BASE.catalogo}/pokemons/${encodeURIComponent(idOuNome)}`),
};
