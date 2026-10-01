import { Project, countWords } from '../types/novelist';

const scene1Prose = `Elena held the seventeenth-century crown glass up to the sodium lamp. Inside the annealing striae, a hairline fracture traced the exact contour of an archipelago no Admiralty chart admitted existed.

Outside the Vault windows, the November tide struck the sea wall in slow, deliberate intervals. Julian set his brass calipers beside her tray of rouge powder and leaned over the slate workbench.

"You're looking at the third grinding mark," Julian said quietly. "Spinoza didn't polish it out. He engraved a latitude."

Elena rotated the bronze bezel fifteen degrees. Where the sodium beam crossed the bevel, three tiny numerals emerged in reverse script: fifty-four degrees, nineteen minutes north. Not a flaw in the cooling furnace—an intentional witness mark hidden inside an instrument commissioned for the Royal Hydrographer in 1674.

"If Maren inventories Drawer Fourteen before the equinox," Elena murmured, setting the lens back into its velvet housing, "she won't just seal the ledger. She'll have the harbour chain raised."`;

const scene2Prose = `Before the bell for night lockup finished its third chime, Archivist Maren stood on the iron gallery above the restoration tables. Her keyring did not rattle; she held the warded bronze key pinched between gloved fingers to silence the brass wards.

"Crate nineteen from the Dogger Bank salvage," Maren called down, her voice carrying cleanly beneath the copper vault. "The Board requires the lens blanks sealed in beeswax before midnight."

Elena kept her palm steady over the chamois cloth. Beneath its folds lay the true objective lens, warm from the lamp housing, while the unground flint blank sat exposed beside the ledger ready for Maren's wax seal.

"The bevel is still cooling from the pitch lap, Madam Archivist," Elena replied without looking up. "Give the balsam twenty minutes to cure or the crown will delaminate in the salt damp."`;

const scene3Prose = `At moonrise they mounted the refractor in the Meridian Tower shutter. Wind off the North Breakwater rattled the copper louvers, carrying freezing spray across the granite sill.

Julian adjusted the counterweight until the brass tube balanced on its trunnions. When the star Fomalhaut crossed the hairline wire, the phantom shoreline resolved in silver relief across the zinc projection plate—seven drowned sea-stacks linked by a submerged causeway leading four leagues west of St. Jude.

"Forty-two minutes at the spring ebb," Julian read from the tidal tables pinned under his compass. "That is all the Atlantic grants us tomorrow night."`;

const scene4Prose = `At the equinoctial spring ebb, the granite survey markers surfaced through the kelp beds for forty-two minutes.

Elena stepped down from the skiff onto slick basalt dressed three centuries ago by Dutch masons. Ahead in the sea fog, the third milestone bore the same astronomical sigil ground into Spinoza's lens.`;

export const INITIAL_PROJECTS: Project[] = [
  {
    id: 1,
    title: 'The Glass Cartographer',
    author: 'Clara Vance',
    genre: 'Literary Speculative Fiction',
    synopsis:
      'A seventeenth-century optical lens discovered in the Old Admiralty Vault reveals a hidden latitude and a submerged tidal causeway that only surfaces during the equinoctial ebb.',
    targetWords: 75000,
    updatedAt: '2026-10-01T04:30:00Z',
    characters: [
      {
        id: 101,
        projectId: 1,
        name: 'Elena Rostova',
        role: 'Protagonist',
        bio: 'Senior restorer of seventeenth-century celestial lenses at the Maritime Archive. Meticulous, observant, protective of historical artifacts.',
      },
      {
        id: 102,
        projectId: 1,
        name: 'Julian Vane',
        role: 'Deuteragonist',
        bio: 'Survey hydrographer dismissed from the Admiralty Board after questioning official soundings off St. Jude.',
      },
      {
        id: 103,
        projectId: 1,
        name: 'Archivist Maren',
        role: 'Antagonist',
        bio: 'Keeper of the Sealed Ledger and custodian of the Admiralty Vault. Enforces the Board’s suppression of pre-1680 charts.',
      },
      {
        id: 104,
        projectId: 1,
        name: 'Captain Hendrik Voort',
        role: 'Supporting',
        bio: 'Skipper of the salvage ketch Westerly who dredged Crate 19 from the Dogger Bank.',
      },
    ],
    locations: [
      {
        id: 201,
        projectId: 1,
        name: 'The Lantern Vault, Old Admiralty',
        description: 'Subterranean copper-domed optical restoration chamber lit by monochromatic sodium lamps and oil refraction prisms.',
      },
      {
        id: 202,
        projectId: 1,
        name: 'Cliffside Meridian Tower',
        description: 'Wind-scoured basalt transit observatory overlooking the North Breakwater and tidal race.',
      },
      {
        id: 203,
        projectId: 1,
        name: 'The Submerged Causeway of St. Jude',
        description: 'Drowned seventeenth-century stone survey road exposed only during the lowest equinoctial spring tide.',
      },
    ],
    acts: [
      {
        id: 11,
        projectId: 1,
        title: 'Act I: The Refracted Meridian',
        sortOrder: 1,
        chapters: [
          {
            id: 111,
            actId: 11,
            title: 'Chapter 1: Salt on the Objective Lens',
            targetWords: 450,
            sortOrder: 1,
            scenes: [
              {
                id: 1111,
                chapterId: 111,
                title: 'Scene 1: The Cracked Astrolabe',
                content: scene1Prose,
                sideNotes:
                  'Establish the sensory contrast between the warm sodium lamp inside the Lantern Vault and the freezing November tide outside.\n\nContinuity note: Spinoza ground lenses in Voorburg between 1663 and 1670.',
                status: 'Completed',
                povCharacterId: 101,
                locationId: 201,
                characterIds: [101, 102],
                targetWords: 200,
                wordCount: countWords(scene1Prose),
                sortOrder: 1,
                updatedAt: '2026-10-01T04:15:00Z',
              },
              {
                id: 1112,
                chapterId: 111,
                title: "Scene 2: Maren's Inventory",
                content: scene2Prose,
                sideNotes:
                  'Keep dialogue terse. Elena swaps the genuine lens with the flint glass blank from Drawer 9 before Maren descends the iron stair.',
                status: 'Edited',
                povCharacterId: 101,
                locationId: 201,
                characterIds: [101, 103],
                targetWords: 220,
                wordCount: countWords(scene2Prose),
                sortOrder: 2,
                updatedAt: '2026-10-01T04:22:00Z',
              },
            ],
          },
          {
            id: 112,
            actId: 11,
            title: 'Chapter 2: The Admiralty Ledger',
            targetWords: 350,
            sortOrder: 2,
            scenes: [
              {
                id: 1121,
                chapterId: 112,
                title: 'Scene 1: Calibration at the Tower',
                content: scene3Prose,
                sideNotes:
                  'Verify astronomical azimuth of Fomalhaut for late November at 54 degrees North latitude.\nInclude Captain Voort delivering the tide almanac.',
                status: 'Drafting',
                povCharacterId: 102,
                locationId: 202,
                characterIds: [101, 102, 104],
                targetWords: 200,
                wordCount: countWords(scene3Prose),
                sortOrder: 1,
                updatedAt: '2026-10-01T04:28:00Z',
              },
            ],
          },
        ],
      },
      {
        id: 12,
        projectId: 1,
        title: 'Act II: Soundings in Amber',
        sortOrder: 2,
        chapters: [
          {
            id: 121,
            actId: 12,
            title: 'Chapter 3: Low Tide at St. Jude',
            targetWords: 400,
            sortOrder: 1,
            scenes: [
              {
                id: 1211,
                chapterId: 121,
                title: 'Scene 1: The Submerged Causeway',
                content: scene4Prose,
                sideNotes:
                  'Midpoint sequence. Need sensory details of kelp, rusted iron mooring rings, and the distant foghorn interval from the breakwater.',
                status: 'Idea',
                povCharacterId: 102,
                locationId: 203,
                characterIds: [101, 102, 103],
                targetWords: 250,
                wordCount: countWords(scene4Prose),
                sortOrder: 1,
                updatedAt: '2026-10-01T04:30:00Z',
              },
            ],
          },
        ],
      },
    ],
  },
];

export function compileProjectMarkdown(project: Project): string {
  const charMap = new Map(project.characters.map((c) => [c.id, c.name]));
  const locMap = new Map(project.locations.map((l) => [l.id, l.name]));

  const lines: string[] = [];
  lines.push(`# ${project.title}`);
  lines.push('');
  if (project.author) {
    lines.push(`**By ${project.author}**  `);
  }
  if (project.genre) {
    lines.push(`*${project.genre}*`);
  }
  lines.push('');
  if (project.synopsis) {
    lines.push(`> ${project.synopsis}`);
    lines.push('');
  }
  lines.push('---');
  lines.push('');

  for (const act of project.acts) {
    lines.push(`# ${act.title}`);
    lines.push('');
    for (const chapter of act.chapters) {
      lines.push(`## ${chapter.title}`);
      lines.push('');
      chapter.scenes.forEach((scene, idx) => {
        lines.push(`### ${scene.title}`);
        lines.push('');
        const meta: string[] = [`Status: ${scene.status}`];
        if (scene.povCharacterId && charMap.has(scene.povCharacterId)) {
          meta.push(`POV: ${charMap.get(scene.povCharacterId)}`);
        }
        if (scene.locationId && locMap.has(scene.locationId)) {
          meta.push(`Setting: ${locMap.get(scene.locationId)}`);
        }
        if (scene.characterIds.length > 0) {
          const castNames = scene.characterIds
            .map((id) => charMap.get(id))
            .filter(Boolean)
            .join(', ');
          if (castNames) meta.push(`Cast: ${castNames}`);
        }
        lines.push(`> *${meta.join(' · ')}*`);
        lines.push('');
        lines.push(scene.content.trim());
        lines.push('');
        if (idx < chapter.scenes.length - 1) {
          lines.push('* * *');
          lines.push('');
        }
      });
    }
  }

  return lines.join('\n');
}

function escapeHtml(raw: string): string {
  return raw
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

export function compileProjectHTML(project: Project): string {
  const charMap = new Map(project.characters.map((c) => [c.id, c.name]));
  const locMap = new Map(project.locations.map((l) => [l.id, l.name]));

  const parts: string[] = [];
  parts.push(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1.0" />
<title>${escapeHtml(project.title)} — Manuscript Export</title>
<style>
  body {
    max-width: 44rem;
    margin: 4rem auto;
    padding: 0 1.5rem;
    font-family: 'Cormorant Garamond', Georgia, serif;
    font-size: 1.15rem;
    line-height: 1.8;
    color: #1c1b18;
    background: #faf9f5;
  }
  header {
    text-align: center;
    margin-bottom: 4rem;
    border-bottom: 1px solid #dcd9d0;
    padding-bottom: 2.5rem;
  }
  h1.book-title { font-size: 2.75rem; margin: 0 0 0.5rem 0; font-weight: 600; }
  .book-meta { font-family: sans-serif; font-size: 0.9rem; color: #68655e; }
  h2.act-title {
    font-size: 1.65rem;
    margin-top: 4rem;
    letter-spacing: 0.04em;
    border-bottom: 1px solid #e5e2d9;
    padding-bottom: 0.5rem;
  }
  h3.chapter-title { font-size: 1.4rem; margin-top: 2.5rem; }
  h4.scene-title { font-size: 1.05rem; color: #57534e; margin: 1.75rem 0 0.35rem 0; font-family: sans-serif; }
  .scene-meta { font-family: sans-serif; font-size: 0.8rem; color: #78716c; margin-bottom: 1.25rem; }
  p { margin: 1rem 0; text-indent: 1.5rem; }
  p.first-para { text-indent: 0; }
  hr.scene-break { border: none; text-align: center; margin: 2.5rem 0; }
  hr.scene-break::after { content: "* * *"; letter-spacing: 0.45rem; color: #78716c; }
</style>
</head>
<body>
<header>
  <h1 class="book-title">${escapeHtml(project.title)}</h1>
  <div class="book-meta">By ${escapeHtml(project.author || 'Anonymous')} &middot; ${escapeHtml(project.genre || 'Novel')}</div>
</header>`);

  for (const act of project.acts) {
    parts.push(`<h2 class="act-title">${escapeHtml(act.title)}</h2>`);
    for (const chapter of act.chapters) {
      parts.push(`<h3 class="chapter-title">${escapeHtml(chapter.title)}</h3>`);
      chapter.scenes.forEach((scene, sIdx) => {
        const meta: string[] = [scene.status];
        if (scene.povCharacterId && charMap.has(scene.povCharacterId)) {
          meta.push(`POV: ${charMap.get(scene.povCharacterId)}`);
        }
        if (scene.locationId && locMap.has(scene.locationId)) {
          meta.push(`Setting: ${locMap.get(scene.locationId)}`);
        }
        parts.push(`<h4 class="scene-title">${escapeHtml(scene.title)}</h4>`);
        parts.push(`<div class="scene-meta">${escapeHtml(meta.join(' · '))}</div>`);
        const paragraphs = scene.content
          .trim()
          .split(/\n\s*\n/)
          .filter(Boolean);
        paragraphs.forEach((p, pIdx) => {
          parts.push(
            `<p class="${pIdx === 0 ? 'first-para' : ''}">${escapeHtml(p.trim())}</p>`
          );
        });
        if (sIdx < chapter.scenes.length - 1) {
          parts.push(`<hr class="scene-break" />`);
        }
      });
    }
  }

  parts.push(`</body>\n</html>`);
  return parts.join('\n');
}

function sqlString(val: string): string {
  return `'${val.replace(/'/g, "''")}'`;
}

export function compileProjectSQLiteDump(project: Project): string {
  const lines: string[] = [];
  lines.push('-- GoNovelist SQLite Data Export');
  lines.push('-- Compatible with gonovelist/schema.sql and database.go');
  lines.push('BEGIN TRANSACTION;');
  lines.push('');
  lines.push(
    `INSERT INTO projects (id, title, author, genre, synopsis, target_words) VALUES (${project.id}, ${sqlString(
      project.title
    )}, ${sqlString(project.author)}, ${sqlString(project.genre)}, ${sqlString(
      project.synopsis
    )}, ${project.targetWords});`
  );
  lines.push('');

  for (const c of project.characters) {
    lines.push(
      `INSERT INTO characters (id, project_id, name, role, bio) VALUES (${c.id}, ${project.id}, ${sqlString(
        c.name
      )}, ${sqlString(c.role)}, ${sqlString(c.bio)});`
    );
  }
  lines.push('');

  for (const l of project.locations) {
    lines.push(
      `INSERT INTO locations (id, project_id, name, description) VALUES (${l.id}, ${project.id}, ${sqlString(
        l.name
      )}, ${sqlString(l.description)});`
    );
  }
  lines.push('');

  for (const act of project.acts) {
    lines.push(
      `INSERT INTO acts (id, project_id, title, sort_order) VALUES (${act.id}, ${project.id}, ${sqlString(
        act.title
      )}, ${act.sortOrder});`
    );
    for (const ch of act.chapters) {
      lines.push(
        `INSERT INTO chapters (id, act_id, title, target_words, sort_order) VALUES (${ch.id}, ${act.id}, ${sqlString(
          ch.title
        )}, ${ch.targetWords}, ${ch.sortOrder});`
      );
      for (const sc of ch.scenes) {
        const pov = sc.povCharacterId !== null ? sc.povCharacterId : 'NULL';
        const loc = sc.locationId !== null ? sc.locationId : 'NULL';
        lines.push(
          `INSERT INTO scenes (id, chapter_id, title, content, side_notes, status, pov_character_id, location_id, target_words, word_count, sort_order) VALUES (${sc.id}, ${ch.id}, ${sqlString(
            sc.title
          )}, ${sqlString(sc.content)}, ${sqlString(sc.sideNotes)}, ${sqlString(
            sc.status
          )}, ${pov}, ${loc}, ${sc.targetWords}, ${sc.wordCount}, ${sc.sortOrder});`
        );
        for (const cid of sc.characterIds) {
          lines.push(
            `INSERT INTO scene_characters (scene_id, character_id) VALUES (${sc.id}, ${cid});`
          );
        }
      }
    }
  }

  lines.push('');
  lines.push('COMMIT;');
  return lines.join('\n');
}
