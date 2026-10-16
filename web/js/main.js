// registra as telas e inicia o roteador.
import { registrar, iniciarRoteador } from './router.js';
import { renderLogin } from './screens/login.js';
import { renderMenu } from './screens/menu.js';
import { renderGacha } from './screens/gacha.js';
import { renderColecao } from './screens/colecao.js';
import { renderSelecaoTime, renderBatalha } from './screens/batalha.js';

registrar('login', renderLogin);
registrar('menu', renderMenu);
registrar('gacha', renderGacha);
registrar('colecao', renderColecao);
registrar('time', renderSelecaoTime);
registrar('batalha', renderBatalha);

iniciarRoteador();
