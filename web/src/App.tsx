import { useState } from 'react';
import { DirectoryDemo } from './features/directory/DirectoryDemo';
import { ScheduleDemo } from './features/schedule/ScheduleDemo';

type DemoSurface = 'agenda' | 'directory';

export default function App() {
  const [activeSurface, setActiveSurface] = useState<DemoSurface>('agenda');
  const activeSurfaceLabel = activeSurface === 'agenda' ? 'Agenda demo' : 'Directorio demo';

  return (
    <main className="page">
      <div className="shell app-shell stack">
        <header className="hero hero-product card">
          <div className="hero-kicker">Demo frontend</div>
          <div className="hero-copy stack-tight">
            <h1>Clinic platform demos</h1>
            <p>
              Una demo simple, clara y presentable: Agenda para operar turnos y Directorio para cargar datos sin meter
              complejidad innecesaria.
            </p>
          </div>

          <div className="hero-summary-grid">
            <article className="summary-tile">
              <span className="summary-label">Superficie activa</span>
              <strong>{activeSurfaceLabel}</strong>
              <small>Separación conceptual clara para mostrar cada flujo.</small>
            </article>
            <article className="summary-tile">
              <span className="summary-label">Experiencia demo</span>
              <strong>Simple y prolija</strong>
              <small>Sin routing pesado ni UI kits, pero con look de producto.</small>
            </article>
          </div>
        </header>

        <section className="surface-switcher card stack-tight" aria-label="Selector de demo">
          <div className="surface-switcher-header">
            <div>
              <h2>Elegí qué querés mostrar</h2>
              <p>Cada tab mantiene su foco: operación diaria en Agenda y carga liviana en Directorio.</p>
            </div>
            <span className="badge neutral">Sin routing complejo</span>
          </div>

          <div className="surface-tabs" role="tablist" aria-label="Superficies demo">
            <button
              type="button"
              role="tab"
              aria-selected={activeSurface === 'agenda'}
              className={`surface-tab${activeSurface === 'agenda' ? ' active' : ''}`}
              onClick={() => setActiveSurface('agenda')}
            >
              <span className="surface-tab-eyebrow">Operación</span>
              <strong>Agenda demo</strong>
              <small>Slots, reservas y cancelaciones.</small>
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={activeSurface === 'directory'}
              className={`surface-tab${activeSurface === 'directory' ? ' active' : ''}`}
              onClick={() => setActiveSurface('directory')}
            >
              <span className="surface-tab-eyebrow">Base demo</span>
              <strong>Directorio demo</strong>
              <small>Alta rápida de pacientes y profesionales.</small>
            </button>
          </div>
        </section>

        {activeSurface === 'agenda' ? <ScheduleDemo /> : <DirectoryDemo />}
      </div>
    </main>
  );
}
