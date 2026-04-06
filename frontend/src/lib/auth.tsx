"use client";

import { createContext, useContext, useEffect, useState, useCallback, type ReactNode } from "react";
import { api, ApiRequestError } from "./api";
import type { User } from "@/types/user";

const PUBLIC_PATHS = ["/", "/login", "/signup", "/forgot-password", "/reset-password", "/banned"];

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (login: string, password: string) => Promise<void>;
  signup: (username: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

interface AuthResponse {
  user: User;
  access_token: string;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Try to refresh on mount to restore session
  useEffect(() => {
    const init = async () => {
      try {
        await api("/v1/auth/refresh", { method: "POST" });
        const data = await api<User>("/v1/auth/me");
        setUser(data);
      } catch (err) {
        // No valid session — redirect to appropriate page if on a protected route
        setUser(null);
        if (typeof window !== "undefined") {
          const path = window.location.pathname;
          const isPublic = PUBLIC_PATHS.some((p) => path === p);
          if (!isPublic) {
            // Check if banned — redirect to /banned page
            if (err instanceof ApiRequestError && err.code === "ACCOUNT_BANNED") {
              window.location.replace("/banned");
            } else {
              window.location.replace("/login");
            }
            return;
          }
        }
      } finally {
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

  const refreshUser = useCallback(async () => {
    try {
      const data = await api<User>("/v1/auth/me");
      setUser(data);
    } catch {
      // Ignore errors
    }
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
        refreshUser,
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
