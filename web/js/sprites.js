// URLs dos sprites do repositório oficial da PokeAPI, com fallback.
// Os ícones da Gen VIII são leves (~2 KB) e servem para coleção e menus;
// os sprites grandes ficam para a batalha e a revelação do gacha.

const RAIZ = 'https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon';
const ICONE = `${RAIZ}/versions/generation-viii/icons`;

export const spriteIcone = (id) => `${ICONE}/${id}.png`;
export const spriteFrente = (id) => `${RAIZ}/${id}.png`;
export const spriteCostas = (id) => `${RAIZ}/back/${id}.png`;

// Imagem com cadeia de fallback: se uma URL falhar, tenta a próxima;
// se todas falharem, mostra uma pokébola.
export function imgPokemon({ id, nome, tipo = 'icone', tamanho = 64, preguicoso = true, classe = '' }) {
  const urls = {
    icone: [spriteIcone(id), spriteFrente(id)],
    frente: [spriteFrente(id), spriteIcone(id)],
    costas: [spriteCostas(id), spriteFrente(id), spriteIcone(id)],
  }[tipo];

  const img = document.createElement('img');
  img.className = `sprite ${classe}`.trim();
  img.width = tamanho;
  img.height = tamanho;
  img.alt = `Sprite de ${nome}`;
  img.decoding = 'async';
  if (preguicoso) img.loading = 'lazy';

  let i = 0;
  img.onerror = () => {
    i++;
    if (i < urls.length) img.src = urls[i];
    else { img.onerror = null; img.src = 'assets/img/pokeball.svg'; }
  };
  img.src = urls[0];
  return img;
}

const precarregados = new Set();
export function precarregarSprites(ids, tipo = 'frente') {
  for (const id of ids) {
    const url = tipo === 'costas' ? spriteCostas(id) : spriteFrente(id);
    if (precarregados.has(url)) continue;
    precarregados.add(url);
    const img = new Image();
    img.src = url;
  }
}
