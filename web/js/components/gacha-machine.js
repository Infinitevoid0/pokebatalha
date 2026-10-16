// máquina de gacha em pixel art (SVG em grade de 4px).

const BOLAS = [
  // x, y, cor da metade de cima
  [52, 60, '#e43b44'], [84, 52, '#0095e9'], [116, 60, '#f7d51d'],
  [40, 88, '#b55088'], [72, 84, '#e43b44'], [104, 88, '#3e8948'], [132, 92, '#0095e9'],
  [56, 112, '#f7d51d'], [88, 116, '#e43b44'], [120, 114, '#b55088'],
];

function bola(x, y, cor, i) {
  return `<g class="gacha-bola" style="animation-delay:${(i % 5) * 120}ms" transform="translate(${x} ${y})">
    <rect x="4" y="0" width="16" height="4" fill="#1a1c2c"/>
    <rect x="0" y="4" width="24" height="16" fill="#1a1c2c"/>
    <rect x="4" y="20" width="16" height="4" fill="#1a1c2c"/>
    <rect x="4" y="4" width="16" height="8" fill="${cor}"/>
    <rect x="4" y="12" width="16" height="8" fill="#f2f2f2"/>
    <rect x="4" y="11" width="16" height="2" fill="#1a1c2c"/>
    <rect x="10" y="10" width="4" height="4" fill="#f2f2f2"/>
    <rect x="6" y="6" width="4" height="2" fill="#fff" opacity=".6"/>
  </g>`;
}

export function criarMaquinaGacha() {
  const el = document.createElement('div');
  el.className = 'gacha-maquina';
  el.innerHTML = `
  <svg viewBox="0 0 200 300" width="200" height="300" shape-rendering="crispEdges" role="img" aria-label="Máquina de gacha cheia de pokébolas">
    <!-- cúpula -->
    <rect x="28" y="24" width="144" height="4" fill="#f2f2f2"/>
    <rect x="20" y="28" width="160" height="4" fill="#f2f2f2"/>
    <rect x="16" y="32" width="4" height="120" fill="#f2f2f2"/>
    <rect x="180" y="32" width="4" height="120" fill="#f2f2f2"/>
    <rect x="20" y="32" width="160" height="120" fill="#8fd3ff" opacity=".18"/>
    <rect x="28" y="36" width="8" height="40" fill="#fff" opacity=".35"/>
    <rect x="36" y="36" width="20" height="6" fill="#fff" opacity=".35"/>
    ${BOLAS.map(([x, y, c], i) => bola(x, y + 20, c, i)).join('')}
    <!-- tampa -->
    <rect x="60" y="8" width="80" height="16" fill="#e43b44"/>
    <rect x="60" y="8" width="80" height="4" fill="#ff8a8f"/>
    <rect x="56" y="20" width="88" height="4" fill="#a82a31"/>
    <!-- base -->
    <rect x="12" y="152" width="176" height="8" fill="#a82a31"/>
    <rect x="16" y="160" width="168" height="124" fill="#e43b44"/>
    <rect x="16" y="160" width="168" height="6" fill="#ff8a8f"/>
    <rect x="12" y="284" width="176" height="12" fill="#1a1c2c"/>
    <rect x="16" y="160" width="6" height="124" fill="#a82a31"/>
    <rect x="178" y="160" width="6" height="124" fill="#a82a31"/>
    <!-- placa -->
    <rect x="40" y="172" width="120" height="20" fill="#1a1c2c"/>
    <rect x="44" y="176" width="112" height="12" fill="#f7d51d"/>
    <rect x="52" y="180" width="8" height="4" fill="#1a1c2c"/><rect x="64" y="180" width="8" height="4" fill="#1a1c2c"/>
    <rect x="76" y="180" width="48" height="4" fill="#1a1c2c"/><rect x="128" y="180" width="20" height="4" fill="#1a1c2c"/>
    <!-- manivela -->
    <g class="gacha-manivela">
      <rect x="84" y="204" width="32" height="32" fill="#c6c6d0"/>
      <rect x="84" y="204" width="32" height="4" fill="#f2f2f2"/>
      <rect x="96" y="196" width="8" height="48" fill="#1a1c2c"/>
      <rect x="92" y="212" width="16" height="16" fill="#e43b44"/>
    </g>
    <!-- saída -->
    <rect x="68" y="252" width="64" height="24" fill="#1a1c2c"/>
    <rect x="72" y="256" width="56" height="16" fill="#333c57"/>
  </svg>
  <div class="gacha-bola-saida" aria-hidden="true"><img src="assets/img/pokeball.svg" width="48" height="48" alt=""></div>
  <div class="gacha-flash" aria-hidden="true"></div>`;
  return el;
}
