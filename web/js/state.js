// estado global com pub/sub; só o ID do jogador vai para o localStorage.

const CHAVE_ID = 'pokebatalha.jogador_id';

function lerStorage() {
  try { return localStorage.getItem(CHAVE_ID); } catch { return null; }
}
function gravarStorage(valor) {
  try {
    if (valor == null) localStorage.removeItem(CHAVE_ID);
    else localStorage.setItem(CHAVE_ID, String(valor));
  } catch { /* storage bloqueado: segue sem lembrar */ }
}

export const Estado = {
  jogador: null,   // { id, nome, moedas, pity }
  colecao: null,   // cache de /colecao até abrir pacote
  batalha: null,   // batalha em andamento
  time: [],        // IDs escolhidos na seleção de time

  idSalvo() { return lerStorage(); },

  setJogador(j) {
    this.jogador = j;
    gravarStorage(j?.id ?? null);
    this.notificar('jogador');
  },

  atualizarMoedas(saldo, pity) {
    if (!this.jogador) return;
    this.jogador.moedas = saldo;
    if (pity) this.jogador.pity = pity;
    this.notificar('jogador');
  },

  invalidarColecao() { this.colecao = null; },

  sair() {
    this.jogador = null;
    this.colecao = null;
    this.batalha = null;
    this.time = [];
    gravarStorage(null);
  },

  _ouvintes: {},
  ouvir(chave, fn) {
    (this._ouvintes[chave] ??= []).push(fn);
    return () => { this._ouvintes[chave] = this._ouvintes[chave].filter((f) => f !== fn); };
  },
  notificar(chave) { this._ouvintes[chave]?.forEach((fn) => fn(this[chave])); },
};
