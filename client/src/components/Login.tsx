import { useState } from "react";
import { useAuth } from "../context/AuthContext";

export function Login() {
  const { login } = useAuth();
  const [uid, setUid] = useState("");
  const [pw, setPw] = useState("");
  const [err, setErr] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    try {
      await login(uid, pw);
    } catch {
      setErr("Login failed");
    }
  };

  return (
    <div style={{ maxWidth: 360, margin: "80px auto" }}>
      <h1>CollabBoard</h1>
      <form onSubmit={handleSubmit}>
        <input
          placeholder="User ID"
          value={uid}
          onChange={(e) => setUid(e.target.value)}
          style={{ display: "block", width: "100%", marginBottom: 8, padding: 8 }}
        />
        <input
          type="password"
          placeholder="Password"
          value={pw}
          onChange={(e) => setPw(e.target.value)}
          style={{ display: "block", width: "100%", marginBottom: 8, padding: 8 }}
        />
        <button type="submit" style={{ width: "100%", padding: 8 }}>
          Login
        </button>
        {err && <p style={{ color: "red" }}>{err}</p>}
      </form>
    </div>
  );
}
