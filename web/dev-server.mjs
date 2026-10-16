// servidor de dev sem dependências: serve web/ e faz o mesmo proxy do nginx.conf.
// uso: node web/dev-server.mjs (porta 8080; mude com PORTA=3000).
// aponta para as portas do compose: catalogo 8081, jogadores 8082, batalha 8083.
import http from 'node:http';
import { readFile, stat } from 'node:fs/promises';
import { extname, join, normalize } from 'node:path';
import { fileURLToPath } from 'node:url';

const RAIZ = fileURLToPath(new URL('.', import.meta.url));
const PORTA = Number(process.env.PORTA || 8080);
const ALVOS = {
  catalogo: process.env.CATALOGO_URL || 'http://localhost:8081',
  jogadores: process.env.JOGADORES_URL || 'http://localhost:8082',
  batalha: process.env.BATALHA_URL || 'http://localhost:8083',
};
const TIPOS = {
  '.html': 'text/html; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.mjs': 'text/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8', '.svg': 'image/svg+xml', '.png': 'image/png', '.ico': 'image/x-icon', '.json': 'application/json',
};

function proxy(req, res, servico, caminho) {
  const destino = new URL(caminho, ALVOS[servico]);
  const p = http.request(destino, { method: req.method, headers: { ...req.headers, host: destino.host } }, (r) => {
    res.writeHead(r.statusCode, r.headers);
    r.pipe(res);
  });
  p.setTimeout(90000, () => p.destroy(new Error('timeout')));
  p.on('error', () => {
    if (!res.headersSent) res.writeHead(502, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ codigo: 'servico_indisponivel', mensagem: `${servico} não respondeu` }));
  });
  req.pipe(p);
}

async function estatico(req, res) {
  const caminho = decodeURIComponent(new URL(req.url, 'http://x').pathname);
  let arquivo = normalize(join(RAIZ, caminho));
  if (!arquivo.startsWith(RAIZ) || caminho.endsWith('nginx.conf')) { res.writeHead(404).end(); return; }
  try {
    if ((await stat(arquivo)).isDirectory()) arquivo = join(arquivo, 'index.html');
  } catch {
    arquivo = join(RAIZ, 'index.html'); // fallback da SPA
  }
  try {
    const dados = await readFile(arquivo);
    res.writeHead(200, { 'Content-Type': TIPOS[extname(arquivo)] || 'application/octet-stream', 'Cache-Control': 'no-cache' });
    res.end(dados);
  } catch {
    res.writeHead(404).end();
  }
}

http.createServer((req, res) => {
  const m = req.url.match(/^\/api\/(catalogo|jogadores|batalha)(\/.*)$/);
  if (m) proxy(req, res, m[1], m[2]);
  else estatico(req, res);
}).listen(PORTA, () => console.log(`PokéBatalha web em http://localhost:${PORTA}`));
