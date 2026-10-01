import { Project, countWords } from '../types/novelist';

const scene1Prose = `Ngọc Liên nâng phiến thủy tinh cổ thế kỷ mười bảy lên trước ngọn đèn dầu lạc. Bên trong lớp pha lê trong vắt, một vết rạn mảnh như sợi tơ bỗng khúc xạ ánh sáng vàng ấm thành hình dáng một quần đảo chưa từng xuất hiện trên bất kỳ tấm bản đồ hàng hải nào của triều đình.

Ngoài hiên gỗ, tiếng nước sông Thu Bồn vỗ nhẹ vào mạn thuyền buôn dưới màn sương đầu thu.

"Cô nhìn kỹ góc lệch của chùm sáng xem," Trần Đình Bách vừa nói vừa đặt chiếc la bàn đồng cũ kỹ lên mặt bàn gỗ lim. "Đó không phải vết nứt ngẫu nhiên khi làm nguội thủy tinh. Người thợ xưa đã cố tình giấu tọa độ vào độ cong của thấu kính."

Ngọc Liên xoay nhẹ vòng đồng quanh thấu kính thêm ba khắc. Bóng tối trên tờ giấy dó trải dưới mặt bàn lập tức tách làm đôi, để lộ ba vạch kinh tuyến giao nhau ngay ngoài khơi Cù Lao Chàm.`;

const scene2Prose = `Cụ Thủ Từ họ Phạm chậm rãi mở chiếc rương gỗ trầm hương khóa đồng. Bên trong không phải vàng bạc mà là những cuộn giấy dó đã ngả màu thời gian, ghi chép nhật ký của đội thương thuyền mất tích tám mươi năm trước.

"Mỗi thấu kính được đúc thành một cặp song sinh," cụ trầm giọng nói, ngón tay gầy guộc chỉ vào dòng chữ Hán Nôm viết bằng mực chu sa. "Một phiến đặt trên đỉnh hải đăng cổ ngoài đảo xa, phiến kia nằm trong tay người hoa tiêu dẫn đường."

Đình Bách cúi xuống sát trang giấy: "Vậy nghĩa là ngọn hải đăng ấy vẫn còn nguyên vẹn dưới lớp sương mù Nam Hải?"

"Chỉ mở lối vào đêm rằm tháng tám khi thủy triều xuống thấp nhất," cụ Thủ Từ khẽ gật đầu, ánh mắt xa xăm nhìn ra làn mưa bụi ngoài mái ngói âm dương.`;

const scene3Prose = `Đình Bách trải tấm hải đồ da dê lên mặt bàn, dùng thước đo góc bằng đồng đối chiếu từng chấm sáng phản chiếu từ thấu kính lên bản vẽ.

Mỗi giao điểm ánh sáng tương ứng với một rạn đá ngầm hình bán nguyệt. Nếu đi lệch chỉ nửa hải lý, con tàu buôn sẽ va phải vách đá dựng đứng dưới lòng biển.

"Chúng ta chỉ có đúng hai canh giờ trước khi gió mùa đông bắc tràn về," Đình Bách đánh dấu điểm neo cuối cùng bằng bút lông chấm mực tàu.`;

const scene4Prose = `Tiếng kéo buồm kẽo kẹt vang lên giữa màn sương đặc quánh tại cửa biển Cửa Đại. Ngọc Liên ôm chặt chiếc hộp gỗ trắc đựng thấu kính trước ngực, lắng nghe tiếng sóng đập dồn dập vào mũi thuyền.

Từ phía chân trời phương đông, một vệt sáng xanh biếc chợt lóe lên giữa tầng mây thấp — tín hiệu hồi đáp từ ngọn hải đăng cổ đã ngủ yên suốt một thế kỷ.`;

export const INITIAL_PROJECTS: Project[] = [
  {
    id: 1,
    title: 'Bản Đồ Thủy Tinh Thành Hội An',
    author: 'Nguyễn Minh Khuê',
    genre: 'Tiểu thuyết Lịch sử & Kỳ ảo',
    synopsis:
      'Một nghệ nhân phục chế cổ vật tại thương cảng Hội An thế kỷ XIX phát hiện tấm hải đồ khắc ẩn trong thấu kính thủy tinh của ngọn hải đăng cổ.',
    targetWords: 45000,
    updatedAt: '2026-10-01T12:00:00Z',
    characters: [
      {
        id: 101,
        projectId: 1,
        name: 'Lê Ngọc Liên',
        role: 'Nhân vật chính',
        description:
          'Nghệ nhân chế tác và phục chế thấu kính tại phố cổ Hội An, am hiểu quang học cổ truyền và thư tịch Hán Nôm.',
      },
      {
        id: 102,
        projectId: 1,
        name: 'Trần Đình Bách',
        role: 'Đồng hành',
        description:
          'Thuyền trưởng tàu buôn từng đi qua vùng biển sương mù Nam Hải, giỏi thiên văn hàng hải và đọc hướng gió.',
      },
      {
        id: 103,
        projectId: 1,
        name: 'Cụ Thủ Từ Họ Phạm',
        role: 'Người dẫn đường',
        description:
          'Người trông coi kho thư tịch cổ tại hội quán, nắm giữ cuốn nhật ký hàng hải bị thất truyền.',
      },
    ],
    locations: [
      {
        id: 201,
        projectId: 1,
        name: 'Xưởng Thủy Tinh Phố Cổ',
        description:
          'Căn gác gỗ nhìn ra sông Thu Bồn với những lò nung pha lê, bàn mài thấu kính và đèn dầu lạc.',
      },
      {
        id: 202,
        projectId: 1,
        name: 'Thư Các Chùa Cầu',
        description:
          'Căn phòng lưu trữ bản đồ hàng hải, la bàn cổ và nhật ký thương thuyền trăm năm.',
      },
      {
        id: 203,
        projectId: 1,
        name: 'Bến Thuyền Cửa Đại',
        description:
          'Cửa biển sương mù nơi sông Thu Bồn đổ ra biển lớn, điểm xuất phát của những chuyến hải trình đêm.',
      },
    ],
    acts: [
      {
        id: 11,
        projectId: 1,
        title: 'Hồi I — Vết Rạn Trong Thấu Kính',
        position: 1,
        chapters: [
          {
            id: 111,
            actId: 11,
            title: 'Chương 1: Ánh Đèn Dầu Bên Sông Thu Bồn',
            position: 1,
            targetWords: 2500,
            scenes: [
              {
                id: 1111,
                chapterId: 111,
                title: 'Cảnh 1: Thấu Kính Khúc Xạ',
                summary:
                  'Ngọc Liên và Đình Bách phát hiện tấm hải đồ ẩn hiện qua ánh đèn xuyên qua thấu kính cổ.',
                content: scene1Prose,
                sideNotes:
                  'Ghi chú: Nhấn mạnh chi tiết mùi dầu thông và tiếng nước sông Thu Bồn. Kiểm tra lại cách tính khắc giờ trên vòng đồng.',
                status: 'Đã biên tập',
                povCharacterId: 101,
                locationId: 201,
                targetWords: 1200,
                wordCount: countWords(scene1Prose),
                position: 1,
                updatedAt: '2026-10-01T11:30:00Z',
                characterIds: [101, 102],
              },
              {
                id: 1112,
                chapterId: 111,
                title: 'Cảnh 2: Cuộn Nhật Ký Bằng Giấy Dó',
                summary:
                  'Cụ Thủ Từ tiết lộ bí mật về cặp thấu kính song sinh và đội thuyền mất tích tám mươi năm trước.',
                content: scene2Prose,
                sideNotes:
                  'Cần bổ sung thêm mô tả về dấu triện bằng chu sa ở trang cuối cuốn nhật ký.',
                status: 'Hoàn thành',
                povCharacterId: 101,
                locationId: 202,
                targetWords: 1200,
                wordCount: countWords(scene2Prose),
                position: 2,
                updatedAt: '2026-10-01T11:45:00Z',
                characterIds: [101, 102, 103],
              },
            ],
          },
          {
            id: 112,
            actId: 11,
            title: 'Chương 2: Mật Mã Trên Bản Đồ Hàng Hải',
            position: 2,
            targetWords: 2500,
            scenes: [
              {
                id: 1121,
                chapterId: 112,
                title: 'Cảnh 1: Đối Chiếu Sao Khuê',
                summary:
                  'Đình Bách tính toán góc phương vị từ bản đồ khúc xạ để tìm luồng lạch qua rạn đá ngầm.',
                content: scene3Prose,
                sideNotes:
                  'Ý tưởng mở rộng: Có người lạ mặt theo dõi xưởng thủy tinh từ bên kia bờ sông.',
                status: 'Đang viết',
                povCharacterId: 102,
                locationId: 201,
                targetWords: 1200,
                wordCount: countWords(scene3Prose),
                position: 1,
                updatedAt: '2026-10-01T11:50:00Z',
                characterIds: [101, 102],
              },
            ],
          },
        ],
      },
      {
        id: 12,
        projectId: 1,
        title: 'Hồi II — Hải Trình Ngoài Sương Mù',
        position: 2,
        chapters: [
          {
            id: 121,
            actId: 12,
            title: 'Chương 3: Con Tàu Khởi Hành Lúc Nửa Đêm',
            position: 1,
            targetWords: 3000,
            scenes: [
              {
                id: 1211,
                chapterId: 121,
                title: 'Cảnh 1: Nhổ Neo Trong Đêm Sương',
                summary:
                  'Con thuyền rời bến Cửa Đại khi thủy triều lên cao và bắt gặp ánh sáng xanh từ đảo xa.',
                content: scene4Prose,
                sideNotes:
                  'Chuẩn bị cao trào cuối Hồi II khi sương mù tách ra để lộ ngọn hải đăng cổ.',
                status: 'Ý tưởng',
                povCharacterId: 101,
                locationId: 203,
                targetWords: 1500,
                wordCount: countWords(scene4Prose),
                position: 1,
                updatedAt: '2026-10-01T12:00:00Z',
                characterIds: [101, 102],
              },
            ],
          },
        ],
      },
    ],
  },
  {
    id: 2,
    title: 'Mùa Gió Chướng Trên Đỉnh Ngự Bình',
    author: 'Nguyễn Minh Khuê',
    genre: 'Trinh thám Cổ trang',
    synopsis:
      'Một vị quan ngự sử trẻ tuổi điều tra vụ mất tích bí ẩn của bản khắc đồng Cửu Đỉnh giữa mùa mưa xứ Huế.',
    targetWords: 60000,
    updatedAt: '2026-10-01T10:00:00Z',
    characters: [
      {
        id: 201,
        projectId: 2,
        name: 'Hoàng Trọng Khiêm',
        role: 'Nhân vật chính',
        description: 'Quan Ngự sử trẻ tuổi tại Kinh đô, sắc sảo và trọng chứng cứ.',
      },
    ],
    locations: [
      {
        id: 301,
        projectId: 2,
        name: 'Tàng Thư Lâu',
        description: 'Tòa lầu lưu trữ châu bản và họa đồ kiến trúc cổ nằm giữa hồ Học Hải.',
      },
    ],
    acts: [
      {
        id: 21,
        projectId: 2,
        title: 'Hồi I — Dấu Ấn Trong Đêm Mưa',
        position: 1,
        chapters: [
          {
            id: 211,
            actId: 21,
            title: 'Chương 1: Bức Mật Thư Ở Tàng Thư Lâu',
            position: 1,
            targetWords: 3000,
            scenes: [
              {
                id: 2111,
                chapterId: 211,
                title: 'Cảnh 1: Vết Mực Chưa Khô',
                summary: 'Trọng Khiêm phát hiện tờ mật thư để lại trên án thư gỗ trắc.',
                content:
                  'Mưa Huế đổ trắng mặt hồ Học Hải. Trọng Khiêm khép cánh cửa gỗ lim của Tàng Thư Lâu, ánh nến soi rõ vết sáp ong vừa mới niêm phong trên phong thư...',
                sideNotes: 'Mô tả kỹ âm thanh tiếng mưa rơi trên mái ngói lưu ly.',
                status: 'Đang viết',
                povCharacterId: 201,
                locationId: 301,
                targetWords: 1500,
                wordCount: 33,
                position: 1,
                updatedAt: '2026-10-01T10:00:00Z',
                characterIds: [201],
              },
            ],
          },
        ],
      },
    ],
  },
];

export function compileProjectMarkdown(project: Project): string {
  const lines: string[] = [];
  lines.push(`# ${project.title}`);
  lines.push('');
  if (project.author) lines.push(`**Tác giả:** ${project.author}  `);
  if (project.genre) lines.push(`**Thể loại:** ${project.genre}`);
  lines.push('');
  if (project.synopsis) {
    lines.push(`> ${project.synopsis}`);
    lines.push('');
  }
  lines.push('---');
  lines.push('');

  for (const act of project.acts) {
    lines.push(`## ${act.title}`);
    lines.push('');
    for (const chapter of act.chapters) {
      lines.push(`### ${chapter.title}`);
      lines.push('');
      chapter.scenes.forEach((scene, idx) => {
        lines.push(`#### ${scene.title}`);
        lines.push('');
        if (scene.content.trim()) {
          lines.push(scene.content.trim());
          lines.push('');
        }
        if (idx < chapter.scenes.length - 1) {
          lines.push('* * *');
          lines.push('');
        }
      });
    }
  }
  return lines.join('\n');
}

export function compileProjectHTML(project: Project): string {
  const esc = (str: string) =>
    str
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');

  const parts: string[] = [];
  parts.push(`<!DOCTYPE html>
<html lang="vi">
<head>
<meta charset="UTF-8">
<title>${esc(project.title)}</title>
<style>
  body { font-family: 'Cormorant Garamond', 'Georgia', serif; max-width: 740px; margin: 3rem auto; padding: 0 1.5rem; color: #1C1B18; background: #FAF7F2; line-height: 1.8; }
  h1 { font-size: 2.5rem; margin-bottom: 0.25rem; }
  .meta { color: #57534E; font-style: italic; margin-bottom: 2rem; }
  h2 { margin-top: 3rem; border-bottom: 1px solid #D6D0C4; padding-bottom: 0.4rem; }
  h3 { margin-top: 2rem; color: #3F3C36; }
  h4 { margin-top: 1.5rem; color: #78716C; font-weight: normal; text-transform: uppercase; letter-spacing: 0.08em; font-size: 0.85rem; }
  p { margin: 1.1rem 0; text-indent: 1.5rem; }
  hr.scene-break { border: none; text-align: center; margin: 2rem 0; }
  hr.scene-break::after { content: "* * *"; color: #78716C; letter-spacing: 0.4em; }
</style>
</head>
<body>`);

  parts.push(`<h1>${esc(project.title)}</h1>`);
  parts.push(
    `<div class="meta">Tác giả: ${esc(project.author || 'Khuyết danh')} &bull; Thể loại: ${esc(project.genre || 'Tiểu thuyết')}</div>`
  );

  for (const act of project.acts) {
    parts.push(`<h2>${esc(act.title)}</h2>`);
    for (const chapter of act.chapters) {
      parts.push(`<h3>${esc(chapter.title)}</h3>`);
      chapter.scenes.forEach((scene, idx) => {
        parts.push(`<h4>${esc(scene.title)}</h4>`);
        const paragraphs = scene.content
          .trim()
          .split(/\n\s*\n/)
          .filter(Boolean);
        for (const p of paragraphs) {
          parts.push(`<p>${esc(p.trim())}</p>`);
        }
        if (idx < chapter.scenes.length - 1) {
          parts.push(`<hr class="scene-break">`);
        }
      });
    }
  }

  parts.push(`</body>\n</html>`);
  return parts.join('\n');
}

export function compileProjectSQLiteDump(project: Project): string {
  const sqlEsc = (s: string) => s.replace(/'/g, "''");
  const lines: string[] = [];
  lines.push('-- Bản sao lưu dữ liệu SQLite từ GoNovelist');
  lines.push('PRAGMA foreign_keys = ON;');
  lines.push('BEGIN TRANSACTION;');
  lines.push('');
  lines.push(
    `INSERT INTO projects (id, title, author, genre, synopsis, target_words) VALUES (${project.id}, '${sqlEsc(project.title)}', '${sqlEsc(project.author)}', '${sqlEsc(project.genre)}', '${sqlEsc(project.synopsis)}', ${project.targetWords});`
  );

  for (const c of project.characters) {
    lines.push(
      `INSERT INTO characters (id, project_id, name, role, description) VALUES (${c.id}, ${project.id}, '${sqlEsc(c.name)}', '${sqlEsc(c.role)}', '${sqlEsc(c.description)}');`
    );
  }
  for (const l of project.locations) {
    lines.push(
      `INSERT INTO locations (id, project_id, name, description) VALUES (${l.id}, ${project.id}, '${sqlEsc(l.name)}', '${sqlEsc(l.description)}');`
    );
  }
  for (const act of project.acts) {
    lines.push(
      `INSERT INTO acts (id, project_id, title, position) VALUES (${act.id}, ${project.id}, '${sqlEsc(act.title)}', ${act.position});`
    );
    for (const ch of act.chapters) {
      lines.push(
        `INSERT INTO chapters (id, act_id, title, position, target_words) VALUES (${ch.id}, ${act.id}, '${sqlEsc(ch.title)}', ${ch.position}, ${ch.targetWords});`
      );
      for (const sc of ch.scenes) {
        const pov = sc.povCharacterId ?? 'NULL';
        const loc = sc.locationId ?? 'NULL';
        lines.push(
          `INSERT INTO scenes (id, chapter_id, title, summary, content, side_notes, status, pov_character_id, location_id, target_words, word_count, position) VALUES (${sc.id}, ${ch.id}, '${sqlEsc(sc.title)}', '${sqlEsc(sc.summary)}', '${sqlEsc(sc.content)}', '${sqlEsc(sc.sideNotes)}', '${sqlEsc(sc.status)}', ${pov}, ${loc}, ${sc.targetWords}, ${sc.wordCount}, ${sc.position});`
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
