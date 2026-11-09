import { readFile, writeFile } from 'node:fs/promises';
import { Resvg } from '@resvg/resvg-js';
const source = new URL('../../build/appicon.svg', import.meta.url);
const output = new URL('../../build/appicon.png', import.meta.url);
const svg = await readFile(source, 'utf8');
await writeFile(output, new Resvg(svg, { fitTo: { mode: 'width', value: 1024 } }).render().asPng());
console.log('Generated 1024px application icon');
