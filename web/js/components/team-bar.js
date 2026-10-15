// Barra do time: um ícone por Pokémon (ativo destacado, desmaiado apagado).
// No rival, mostra também nome, tier e poder de todos, não só do ativo.
import { exibir, escapar } from '../utils.js';
import { imgPokemon } from '../sprites.js';

export function criarTeamBar(lado, { detalhado = false } = {}) {
  const el = document.createElement('ul');
  el.className = `team-bar${detalhado ? ' detalhado' : ''}`;
  lado.time.forEach((l, i) => {
    const desmaiado = l.hp <= 0;
    const li = document.createElement('li');
    li.className = `tb-item tier-${l.tier}${i === lado.ativo ? ' ativo' : ''}${desmaiado ? ' desmaiado' : ''}`;
    li.title = `${exibir(l.nome)} — HP ${l.hp}/${l.hp_max}${desmaiado ? ' (desmaiado)' : ''}`;
    li.appendChild(imgPokemon({ id: l.pokemon_id, nome: exibir(l.nome), tamanho: 40, preguicoso: false }));
    if (detalhado) {
      li.insertAdjacentHTML('beforeend', `
        <span class="tb-info">
          <span class="tb-nome">${escapar(exibir(l.nome))}</span>
          <span class="tb-poder">${escapar(l.tier)} · ${l.poder}</span>
        </span>`);
    }
    el.appendChild(li);
  });
  return el;
}
