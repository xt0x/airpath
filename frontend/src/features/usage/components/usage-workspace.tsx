"use client";

import { AppSidebar } from "@/components/app-sidebar";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";

import { UsageLimitsScreen } from "@/features/usage/components/usage-limits-screen";

import styles from "@/features/usage/components/usage-workspace.module.css";

export function UsageWorkspace() {
  return (
    <SidebarProvider className={`app-shell--fixed usage-sidebar-layout ${styles.styleScope}`}>
      <AppSidebar activePath="/usage" />
      <SidebarInset className="usage-workspace__inset">
        <header className="app-shell__toolbar usage-workspace__toolbar">
          <SidebarTrigger className="app-shell__sidebar-trigger usage-workspace__sidebar-trigger -ml-1" />
        </header>
        <UsageLimitsScreen />
      </SidebarInset>
    </SidebarProvider>
  );
}
