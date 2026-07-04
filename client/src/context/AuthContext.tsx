import { createContext, useContext, useState, useEffect, type ReactNode } from "react";
import { isAuthenticated, login as apiLogin, logout as apiLogout } from "../api/auth";

interface AuthContextValue {
  userId: string;
  isAuth: boolean;
  login: (userId: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [userId, setUserId] = useState(() => localStorage.getItem("user_id") || "");
  const [isAuth, setIsAuth] = useState(isAuthenticated);

  useEffect(() => {
    setIsAuth(isAuthenticated());
  }, [userId]);

  const login = async (uid: string, password: string) => {
    await apiLogin(uid, password);
    localStorage.setItem("user_id", uid);
    setUserId(uid);
  };

  const logout = () => {
    apiLogout();
    localStorage.removeItem("user_id");
    setUserId("");
  };

  return (
    <AuthContext.Provider value={{ userId, isAuth, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth outside AuthProvider");
  return ctx;
}
