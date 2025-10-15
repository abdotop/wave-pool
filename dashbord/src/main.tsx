import { render } from "preact";
import "./index.css";
import { api } from "./lib/api";
import { user } from "./lib/session";
import wavePollLogo from "/wave_pool.svg";
import { LoginPage } from "./pages/LoginPage";
import { DeveloperPortal } from "./pages/DeveloperPortalPage";

const health = api["GET/api/health"].signal();
health.fetch();

export function App() {
  if (health.pending) {
    return <div>Loading...</div>;
  }
  if (!health.data) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-base-100 flex-col gap-4">
        <img src={wavePollLogo} alt="Wave Pool Logo" />
        <h1 className="text-2xl font-semibold text-base-content">
          The Wave Pool API is down
        </h1>
      </div>
    );
  }
  if (!user.data) {
    return <LoginPage />;
  }
  return <DeveloperPortal />;
}

render(<App />, document.getElementById("app")!);
