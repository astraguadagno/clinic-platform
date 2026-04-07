import { useState } from 'react';
import { DirectoryDemo } from './features/directory/DirectoryDemo';
import { ScheduleDemo } from './features/schedule/ScheduleDemo';

type DemoSurface = 'agenda' | 'directory';

export default function App() {
  const [activeSurface, setActiveSurface] = useState<DemoSurface>('agenda');
  const activeSurfaceLabel = activeSurface === 'agenda' ? 'Agenda' : 'Directorio';

  return (
    <main className="page">
      <div className="shell app-shell stack">
        <header className="hero hero-product card">
          <div className="hero-kicker">Demo frontend</div>
          <div className="hero-copy stack-tight">
            <h1>Clinic platform demos</h1>
            <p>
              Una demo simple y consistente: Agenda para operar turnos y Directorio para cargar la base sin mezclar
              contextos.
            </p>
          </div>

          <div className="hero-summary-grid">
            <article className="summary-tile">
              <span className="summary-label">Superficie activa</span>
              <strong>{activeSurfaceLabel}</strong>
              <small>Cada vista conserva su propio foco.</small>
            </article>
            <article className="summary-tile">
              <span className="summary-label">Objetivo</span>
              <strong>Demo clara</strong>
              <small>Menos ruido visual, más legibilidad operativa.</small>
            </article>
          </div>
        </header>

        <section className="surface-switcher card stack-tight" aria-label="Selector de demo">
          <div className="surface-switcher-header">
            <div>
              <h2>Elegí la superficie</h2>
              <p>Agenda prioriza la operación diaria. Directorio mantiene la carga de datos aparte.</p>
            </div>
            <span className="badge neutral">Demo liviana</span>
          </div>

          <div className="surface-tabs" role="tablist" aria-label="Superficies demo">
            <button
              type="button"
              role="tab"
              aria-selected={activeSurface === 'agenda'}
              className={`surface-tab${activeSurface === 'agenda' ? ' active' : ''}`}
              onClick={() => setActiveSurface('agenda')}
            >
              <span className="surface-tab-eyebrow">Operación diaria</span>
              <strong>Agenda</strong>
              <small>Slots, reservas y cancelaciones.</small>
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={activeSurface === 'directory'}
              className={`surface-tab${activeSurface === 'directory' ? ' active' : ''}`}
              onClick={() => setActiveSurface('directory')}
            >
              <span className="surface-tab-eyebrow">Carga base</span>
              <strong>Directorio</strong>
              <small>Alta rápida de pacientes y profesionales.</small>
            </button>
          </div>
        </section>

        {activeSurface === 'agenda' ? <ScheduleDemo /> : <DirectoryDemo />}
      </div>
    </main>
  );
}
