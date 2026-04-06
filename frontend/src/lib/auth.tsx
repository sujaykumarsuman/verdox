"use client";

import { createContext, useContext, useEffect, useState, useCallback, type ReactNode } from "react";
import { api } from "./api";
import type { User } from "@/types/user";

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (login: string, password: string) => Promise<void>;
  signup: (username: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

interface AuthResponse {
  user: User;
  access_token: string;
}

interface TokenResponse {
  access_token: string;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Try to refresh on mount to restore session
  useEffect(() => {
    const init = async () => {
      try {
        await api<TokenResponse>("/v1/auth/refresh", { method: "POST" });
        // If refresh succeeded, we need to get user info
        // The refresh endpoint only returns access_token, not user data
        // For now, we'll decode from the cookie or make a separate call
        // Since we don't have a /me endpoint yet, we'll set a flag
        setIsLoading(false);
      } catch {
        setIsLoading(false);
      }
    };
    init();
  }, []);

  const login = useCallback(async (loginStr: string, password: string) => {
    const data = await api<AuthResponse>("/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ login: loginStr, password }),
    });
    setUser(data.user);
  }, []);

  const signup = useCallback(async (username: string, email: string, password: string) => {
    const data = await api<AuthResponse>("/v1/auth/signup", {
      method: "POST",
      body: JSON.stringify({ username, email, password }),
    });
    setUser(data.user);
  }, []);

  const logout = useCallback(async () => {
    try {
      await api("/v1/auth/logout", { method: "POST" });
    } catch {
      // Ignore errors during logout
    }
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        isAuthenticated: !!user,
        login,
        signup,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
