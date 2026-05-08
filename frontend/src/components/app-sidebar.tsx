"use client";

import * as React from "react";
import { GalleryVerticalEnd, Gauge, Map, Plane, Search, Settings2 } from "lucide-react";

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from "@/components/ui/sidebar";
import { cn } from "@/lib/utils";

const APP_SIDEBAR_MENU_BUTTON_CLASS_NAME =
  "font-normal group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:p-0! group-data-[collapsible=icon]:overflow-visible! group-data-[collapsible=icon]:bg-transparent";
const APP_SIDEBAR_IDENTITY_BUTTON_CLASS_NAME = cn(
  APP_SIDEBAR_MENU_BUTTON_CLASS_NAME,
  "hover:bg-transparent active:bg-transparent group-data-[collapsible=icon]:rounded-[4px]! group-data-[collapsible=icon]:overflow-hidden!",
);
const APP_SIDEBAR_COLLAPSED_LABEL_CLASS_NAME =
  "text-sm leading-5 font-normal group-data-[collapsible=icon]:hidden";

function blurAfterPointerActivation(event: React.MouseEvent<HTMLButtonElement>) {
  if (event.detail > 0) {
    event.currentTarget.blur();
  }
}

const data = {
  identity: {
    name: "Airpath Ops",
    logo: GalleryVerticalEnd,
    plan: "Operations",
  },
  navigationSections: [
    {
      label: "Workspace",
      items: [
        {
          title: "Live Map",
          url: "/",
          icon: Map,
        },
        {
          title: "Flight Search",
          url: "/?flightSearch=1",
          icon: Search,
          action: "flight-search" as const,
        },
        {
          title: "Tracked Flights",
          url: "/?trackedFlights=1",
          icon: Plane,
          action: "tracked-flights" as const,
        },
      ],
    },
    {
      label: "Data",
      items: [
        {
          title: "Usage & Limits",
          url: "/usage",
          icon: Gauge,
        },
      ],
    },
    {
      label: "System",
      items: [
        {
          title: "Settings",
          url: "#",
          icon: Settings2,
        },
      ],
    },
  ],
};

function AppIdentity({
  identity,
}: {
  identity: {
    name: string;
    logo: React.ElementType;
    plan: string;
  };
}) {
  const IdentityLogo = identity.logo;

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton size="lg" className={APP_SIDEBAR_IDENTITY_BUTTON_CLASS_NAME}>
          <div className="flex aspect-square size-8 items-center justify-center rounded-[4px] bg-sidebar-primary text-sidebar-primary-foreground group-data-[collapsible=icon]:size-8! group-data-[collapsible=icon]:rounded-[4px]!">
            <IdentityLogo className="size-4" />
          </div>
          <div
            className={cn(
              "grid flex-1 text-left text-sm leading-tight",
              APP_SIDEBAR_COLLAPSED_LABEL_CLASS_NAME,
            )}
          >
            <span className="truncate font-medium">{identity.name}</span>
            <span className="truncate text-xs">{identity.plan}</span>
          </div>
        </SidebarMenuButton>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}

function NavSections({
  sections,
  activePath,
  onFlightSearchSelect,
  onTrackedFlightsSelect,
}: {
  sections: {
    label: string;
    items: {
      title: string;
      url: string;
      icon: React.ElementType;
      action?: "flight-search" | "tracked-flights";
    }[];
  }[];
  activePath?: string;
  onFlightSearchSelect?: () => void;
  onTrackedFlightsSelect?: () => void;
}) {
  return (
    <>
      {sections.map((section) => (
        <SidebarGroup key={section.label}>
          <SidebarGroupLabel>{section.label}</SidebarGroupLabel>
          <SidebarMenu>
            {section.items.map((item) => (
              <SidebarMenuItem key={item.title}>
                {item.action === "flight-search" && onFlightSearchSelect !== undefined ? (
                  <SidebarMenuButton
                    asChild
                    tooltip={item.title}
                    isActive={activePath === item.action}
                    className={APP_SIDEBAR_MENU_BUTTON_CLASS_NAME}
                  >
                    <button
                      type="button"
                      data-flight-search-trigger="true"
                      onClick={(event) => {
                        blurAfterPointerActivation(event);
                        onFlightSearchSelect();
                      }}
                    >
                      <item.icon />
                      <span className={APP_SIDEBAR_COLLAPSED_LABEL_CLASS_NAME}>{item.title}</span>
                    </button>
                  </SidebarMenuButton>
                ) : item.action === "tracked-flights" && onTrackedFlightsSelect !== undefined ? (
                  <SidebarMenuButton
                    asChild
                    tooltip={item.title}
                    isActive={activePath === item.action}
                    className={APP_SIDEBAR_MENU_BUTTON_CLASS_NAME}
                  >
                    <button
                      type="button"
                      data-tracked-flights-trigger="true"
                      onClick={(event) => {
                        blurAfterPointerActivation(event);
                        onTrackedFlightsSelect();
                      }}
                    >
                      <item.icon />
                      <span className={APP_SIDEBAR_COLLAPSED_LABEL_CLASS_NAME}>{item.title}</span>
                    </button>
                  </SidebarMenuButton>
                ) : (
                  <SidebarMenuButton
                    asChild
                    tooltip={item.title}
                    isActive={activePath === item.url}
                    className={APP_SIDEBAR_MENU_BUTTON_CLASS_NAME}
                  >
                    <a href={item.url}>
                      <item.icon />
                      <span className={APP_SIDEBAR_COLLAPSED_LABEL_CLASS_NAME}>{item.title}</span>
                    </a>
                  </SidebarMenuButton>
                )}
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarGroup>
      ))}
    </>
  );
}

export function AppSidebar({
  activePath,
  onFlightSearchSelect,
  onTrackedFlightsSelect,
  ...props
}: React.ComponentProps<typeof Sidebar> & {
  activePath?: string;
  onFlightSearchSelect?: () => void;
  onTrackedFlightsSelect?: () => void;
}) {
  return (
    <Sidebar {...props} collapsible="icon">
      <SidebarHeader>
        <AppIdentity identity={data.identity} />
      </SidebarHeader>
      <SidebarContent>
        <NavSections
          sections={data.navigationSections}
          activePath={activePath}
          onFlightSearchSelect={onFlightSearchSelect}
          onTrackedFlightsSelect={onTrackedFlightsSelect}
        />
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  );
}
