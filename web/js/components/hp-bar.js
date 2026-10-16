// barra de HP: verde > 50%, amarela até 25%, vermelha abaixo; anima com scaleX.

function corHP(fracao) {
  if (fracao > 0.5) return 'var(--hp-alto)';
  if (fracao > 0.25) return 'var(--hp-medio)';
  return 'var(--hp-baixo)';
}

export function criarHPBar(hp, max) {
  const el = document.createElement('div');
  el.className = 'hp';
  el.innerHTML = `
    <span class="hp-rotulo">HP</span>
    <div class="hp-trilho" role="progressbar" aria-valuemin="0">
      <div class="hp-preenchimento"></div>
    </div>
    <span class="hp-texto"></span>`;
  atualizarHPBar(el, hp, max, false);
  return el;
}

export function atualizarHPBar(el, hp, max, animar = true) {
  const fracao = max > 0 ? Math.max(0, Math.min(hp / max, 1)) : 0;
  const barra = el.querySelector('.hp-preenchimento');
  if (!animar) barra.style.transition = 'none';
  barra.style.transform = `scaleX(${fracao})`;
  barra.style.setProperty('--cor-hp', corHP(fracao));
  if (!animar) requestAnimationFrame(() => { barra.style.transition = ''; });
  const trilho = el.querySelector('.hp-trilho');
  trilho.setAttribute('aria-valuemax', String(max));
  trilho.setAttribute('aria-valuenow', String(hp));
  trilho.setAttribute('aria-label', `HP ${hp} de ${max}`);
  el.querySelector('.hp-texto').textContent = `${hp}/${max}`;
}
