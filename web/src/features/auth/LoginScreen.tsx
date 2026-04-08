import { useState } from 'react';

type LoginScreenProps = {
  errorMessage: string;
  isSubmitting: boolean;
  onLogin: (email: string, password: string) => Promise<void>;
};

export function LoginScreen({ errorMessage, isSubmitting, onLogin }: LoginScreenProps) {
  const [email, setEmail] = useState('admin@clinic.local');
  const [password, setPassword] = useState('admin123');

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await onLogin(email, password);
  }

  return (
    <main className="page page-centered auth-page">
      <div className="shell auth-shell">
        <section className="card auth-card stack">
          <div className="hero-kicker">Sprint 1 · acceso mínimo</div>
          <div className="stack-tight">
            <h1>Ingresá a Clinic Platform</h1>
            <p>
              Login simple contra <code>/directory-api/auth/login</code> con restauración de sesión usando{' '}
              <code>/directory-api/auth/me</code>.
            </p>
          </div>

          <form className="stack" onSubmit={handleSubmit}>
            <div className="field">
              <label htmlFor="login-email">Email</label>
              <input
                id="login-email"
                type="email"
                autoComplete="username"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                placeholder="admin@clinic.local"
              />
            </div>

            <div className="field">
              <label htmlFor="login-password">Contraseña</label>
              <input
                id="login-password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder="••••••••"
              />
            </div>

            <div className="toolbar auth-toolbar">
              <button className="button" type="submit" disabled={isSubmitting}>
                {isSubmitting ? 'Ingresando...' : 'Iniciar sesión'}
              </button>
              <span className="helper helper-inline">Seeds demo: admin/admin123 · secretary/secretary123 · doctor/doctor123</span>
            </div>

            {errorMessage ? <div className="inline-note inline-note-error">{errorMessage}</div> : null}
          </form>

          <div className="auth-footnotes">
            <span className="badge neutral">Token guardado en localStorage para este sprint</span>
            <span className="badge info">Sin router ni state manager extra</span>
          </div>
        </section>
      </div>
    </main>
  );
}
