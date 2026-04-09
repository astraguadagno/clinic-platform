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
          <div className="hero-kicker">Acceso seguro</div>
          <div className="stack-tight">
            <h1>Ingresá a tu espacio de trabajo</h1>
            <p>Usá tu usuario para entrar a la operación diaria de la clínica.</p>
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
                {isSubmitting ? 'Ingresando...' : 'Ingresar'}
              </button>
              <span className="helper helper-inline">
                Credenciales disponibles para esta instancia: admin/admin123 · secretary/secretary123 · doctor/doctor123
              </span>
            </div>

            {errorMessage ? <div className="inline-note inline-note-error">{errorMessage}</div> : null}
          </form>

          <div className="auth-footnotes">
            <span className="badge neutral">La sesión queda abierta en este navegador</span>
            <span className="badge info">Tu acceso define qué espacios vas a ver</span>
          </div>
        </section>
      </div>
    </main>
  );
}
