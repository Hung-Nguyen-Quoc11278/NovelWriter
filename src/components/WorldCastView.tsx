import React, { useState } from 'react';
import { Plus, Trash2, ArrowUpRight } from 'lucide-react';
import { Character, Location, Project } from '../types/novelist';

interface WorldCastViewProps {
  project: Project;
  onAddCharacter: (name: string, role: Character['role'], bio: string) => void;
  onDeleteCharacter: (id: number) => void;
  onAddLocation: (name: string, description: string) => void;
  onDeleteLocation: (id: number) => void;
  onSelectScene: (sceneId: number) => void;
}

export const WorldCastView: React.FC<WorldCastViewProps> = ({
  project,
  onAddCharacter,
  onDeleteCharacter,
  onAddLocation,
  onDeleteLocation,
  onSelectScene,
}) => {
  const [charName, setCharName] = useState('');
  const [charRole, setCharRole] = useState<Character['role']>('Supporting');
  const [charBio, setCharBio] = useState('');

  const [locName, setLocName] = useState('');
  const [locDesc, setLocDesc] = useState('');

  const allScenes = project.acts.flatMap((a) =>
    a.chapters.flatMap((c) =>
      c.scenes.map((s) => ({
        ...s,
        actTitle: a.title,
        chapterTitle: c.title,
      }))
    )
  );

  const handleCreateChar = (e: React.FormEvent) => {
    e.preventDefault();
    if (!charName.trim()) return;
    onAddCharacter(charName.trim(), charRole, charBio.trim());
    setCharName('');
    setCharBio('');
  };

  const handleCreateLoc = (e: React.FormEvent) => {
    e.preventDefault();
    if (!locName.trim()) return;
    onAddLocation(locName.trim(), locDesc.trim());
    setLocName('');
    setLocDesc('');
  };

  return (
    <div className="max-w-6xl mx-auto px-6 py-8 space-y-12">
      <div className="border-b border-[#E6E4DD] pb-6 flex flex-col md:flex-row md:items-end justify-between gap-4">
        <div>
          <p className="text-xs text-[#68655E] mb-1">
            Context &amp; Metadata Registry · {project.title}
          </p>
          <h1 className="text-3xl font-semibold text-[#1C1B18] text-balance">
            Dramatis Personae &amp; World Settings
          </h1>
        </div>
        <div className="text-xs text-[#68655E] font-mono tabular-nums">
          <span>{project.characters.length} Cast Members</span>
          <span className="mx-2" aria-hidden="true">·</span>
          <span>{project.locations.length} Locations</span>
          <span className="mx-2" aria-hidden="true">·</span>
          <span>{allScenes.length} Total Scenes</span>
        </div>
      </div>

      {/* Characters Section */}
      <section className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
        <div className="lg:col-span-4 border border-[#E6E4DD] bg-[#F1EFEA]/60 rounded-xl p-5">
          <h2 className="text-xl font-semibold text-[#1C1B18] mb-1">
            Register Cast Character
          </h2>
          <p className="text-xs text-[#68655E] mb-4">
            Characters added here become selectable as Scene POV or scene participants.
          </p>
          <form onSubmit={handleCreateChar} className="space-y-3.5">
            <div>
              <label className="block text-xs font-medium text-[#1C1B18] mb-1">
                Full Name
              </label>
              <input
                type="text"
                value={charName}
                onChange={(e) => setCharName(e.target.value)}
                placeholder="e.g., Sister Beatrice Vane"
                className="w-full px-3 py-2 text-sm bg-white border border-[#DCD9D0] rounded-lg focus:outline-none focus:border-[#1E3A5F]"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-[#1C1B18] mb-1">
                Narrative Role
              </label>
              <select
                value={charRole}
                onChange={(e) => setCharRole(e.target.value as Character['role'])}
                className="w-full px-3 py-2 text-sm bg-white border border-[#DCD9D0] rounded-lg focus:outline-none focus:border-[#1E3A5F]"
              >
                <option value="Protagonist">Protagonist</option>
                <option value="Deuteragonist">Deuteragonist</option>
                <option value="Antagonist">Antagonist</option>
                <option value="Supporting">Supporting</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-[#1C1B18] mb-1">
                Character Dossier &amp; Voice Notes
              </label>
              <textarea
                rows={3}
                value={charBio}
                onChange={(e) => setCharBio(e.target.value)}
                placeholder="Motivations, speech cadence, physical details..."
                className="w-full px-3 py-2 text-sm bg-white border border-[#DCD9D0] rounded-lg focus:outline-none focus:border-[#1E3A5F]"
              />
            </div>
            <button
              type="submit"
              className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 text-xs font-semibold text-white bg-[#1E3A5F] rounded-lg hover:bg-[#162B47] transition-colors whitespace-nowrap"
            >
              <Plus className="w-3.5 h-3.5" />
              Add to Cast Registry
            </button>
          </form>
        </div>

        <div className="lg:col-span-8 space-y-4">
          {project.characters.map((char) => {
            const povScenes = allScenes.filter((s) => s.povCharacterId === char.id);
            const presentScenes = allScenes.filter((s) =>
              s.characterIds.includes(char.id)
            );

            return (
              <div
                key={char.id}
                className="border border-[#E6E4DD] bg-white rounded-xl p-5 flex flex-col justify-between gap-4"
              >
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <h3 className="text-lg font-semibold text-[#1C1B18]">
                      {char.name}
                    </h3>
                    <div className="flex items-center gap-2 text-xs text-[#68655E] mt-0.5 font-mono tabular-nums">
                      <span>{char.role}</span>
                      <span aria-hidden="true">·</span>
                      <span>POV in {povScenes.length} scenes</span>
                      <span aria-hidden="true">·</span>
                      <span>Present in {presentScenes.length} scenes</span>
                    </div>
                  </div>
                  <button
                    onClick={() => onDeleteCharacter(char.id)}
                    title="Remove character"
                    className="p-1.5 text-[#68655E] hover:text-red-700 transition-colors"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>

                {char.bio && (
                  <p className="text-sm text-[#3A3832] leading-relaxed">
                    {char.bio}
                  </p>
                )}

                {presentScenes.length > 0 && (
                  <div className="pt-3 border-t border-[#F1EFEA] flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs">
                    <span className="text-[#68655E] font-medium">Scene Appearances:</span>
                    {presentScenes.map((sc) => (
                      <button
                        key={sc.id}
                        onClick={() => onSelectScene(sc.id)}
                        className="inline-flex items-center gap-1 text-[#1E3A5F] hover:underline font-medium whitespace-nowrap"
                      >
                        <span>{sc.title}</span>
                        <ArrowUpRight className="w-3 h-3" />
                      </button>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </section>

      {/* Locations Section */}
      <section className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start pt-6 border-t border-[#E6E4DD]">
        <div className="lg:col-span-4 border border-[#E6E4DD] bg-[#F1EFEA]/60 rounded-xl p-5">
          <h2 className="text-xl font-semibold text-[#1C1B18] mb-1">
            Register World Location
          </h2>
          <p className="text-xs text-[#68655E] mb-4">
            Settings and architectural backdrops mapped to individual scenes.
          </p>
          <form onSubmit={handleCreateLoc} className="space-y-3.5">
            <div>
              <label className="block text-xs font-medium text-[#1C1B18] mb-1">
                Location / Setting Name
              </label>
              <input
                type="text"
                value={locName}
                onChange={(e) => setLocName(e.target.value)}
                placeholder="e.g., The Submerged Breakwater"
                className="w-full px-3 py-2 text-sm bg-white border border-[#DCD9D0] rounded-lg focus:outline-none focus:border-[#1E3A5F]"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-[#1C1B18] mb-1">
                Sensory &amp; Architectural Notes
              </label>
              <textarea
                rows={3}
                value={locDesc}
                onChange={(e) => setLocDesc(e.target.value)}
                placeholder="Lighting, acoustics, weather, historical context..."
                className="w-full px-3 py-2 text-sm bg-white border border-[#DCD9D0] rounded-lg focus:outline-none focus:border-[#1E3A5F]"
              />
            </div>
            <button
              type="submit"
              className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 text-xs font-semibold text-white bg-[#1E3A5F] rounded-lg hover:bg-[#162B47] transition-colors whitespace-nowrap"
            >
              <Plus className="w-3.5 h-3.5" />
              Add to Location Atlas
            </button>
          </form>
        </div>

        <div className="lg:col-span-8 space-y-4">
          {project.locations.map((loc) => {
            const linkedScenes = allScenes.filter((s) => s.locationId === loc.id);
            return (
              <div
                key={loc.id}
                className="border border-[#E6E4DD] bg-white rounded-xl p-5 flex flex-col justify-between gap-4"
              >
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <h3 className="text-lg font-semibold text-[#1C1B18]">
                      {loc.name}
                    </h3>
                    <div className="text-xs text-[#68655E] mt-0.5 font-mono tabular-nums">
                      Active setting in {linkedScenes.length} scenes
                    </div>
                  </div>
                  <button
                    onClick={() => onDeleteLocation(loc.id)}
                    title="Remove location"
                    className="p-1.5 text-[#68655E] hover:text-red-700 transition-colors"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>

                {loc.description && (
                  <p className="text-sm text-[#3A3832] leading-relaxed">
                    {loc.description}
                  </p>
                )}

                {linkedScenes.length > 0 && (
                  <div className="pt-3 border-t border-[#F1EFEA] flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs">
                    <span className="text-[#68655E] font-medium">Scenes Set Here:</span>
                    {linkedScenes.map((sc) => (
                      <button
                        key={sc.id}
                        onClick={() => onSelectScene(sc.id)}
                        className="inline-flex items-center gap-1 text-[#1E3A5F] hover:underline font-medium whitespace-nowrap"
                      >
                        <span>{sc.title}</span>
                        <ArrowUpRight className="w-3 h-3" />
                      </button>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </section>
    </div>
  );
};
