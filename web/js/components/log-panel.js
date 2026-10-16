// log da batalha digitado linha a linha; clicar acelera.
import { espera } from '../utils.js';

export function criarLogPanel() {
  const el = document.createElement('div');
  el.className = 'log-panel dialogo';
  el.setAttribute('role', 'log');
  el.setAttribute('aria-label', 'Log da batalha');
  el.innerHTML = '<ol class="log-linhas"></ol><span class="seta" aria-hidden="true" hidden>▼</span>';
  let rapido = false;
  el.addEventListener('click', () => { rapido = true; });

  return {
    el,
    limpar() { el.querySelector('.log-linhas').innerHTML = ''; },
    async escrever(texto, classe = '') {
      const lista = el.querySelector('.log-linhas');
      const li = document.createElement('li');
      if (classe) li.className = classe;
      lista.appendChild(li);
      for (let i = 0; i < texto.length; i++) {
        if (rapido) { li.textContent = texto; break; }
        li.textContent += texto[i];
        if (texto[i] !== ' ') await espera(16);
        if (i % 8 === 0) el.scrollTop = el.scrollHeight;
      }
      el.scrollTop = el.scrollHeight;
      if (!rapido) await espera(260);
    },
    // velocidade normal a cada turno.
    reiniciarVelocidade() { rapido = false; },
    seta(mostrar) { el.querySelector('.seta').hidden = !mostrar; },
  };
}
