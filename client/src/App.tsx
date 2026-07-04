import { AuthProvider, useAuth } from "./context/AuthContext";
import { Login } from "./components/Login";
import { Board } from "./components/Board";

function AppInner() {
  const { isAuth, logout } = useAuth();

  if (!isAuth) return <Login />;

  const boardId = new URLSearchParams(location.search).get("board_id") || "default";

  return (
    <div>
      <div style={{ position: "absolute", top: 8, right: 8 }}>
        <button onClick={logout} style={{ fontSize: 12, padding: "4px 8px" }}>
          Logout
        </button>
      </div>
      <Board boardId={boardId} />
    </div>
  );
}

export function App() {
  return (
    <AuthProvider>
      <AppInner />
    </AuthProvider>
  );
}
