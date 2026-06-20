"use client";

import Icon from "@/components/Icon";
import Link from "next/link";
import type { ComponentProps, ReactNode } from "react";
import {
  PortalField as KitPortalField,
  PortalInput as KitPortalInput,
  PortalSelect as KitPortalSelect,
  PortalSection as KitPortalSection,
  PortalActions as KitPortalActions,
  SelectionCard as KitSelectionCard,
  DataList as KitDataList,
  DataListRow as KitDataListRow,
  HubCard as KitHubCard,
  FormAlert as KitFormAlert,
} from "@pegasusx/ui-kit/portal";

export { KitPortalField as PortalField, KitPortalInput as PortalInput, KitPortalSelect as PortalSelect };
export const DataList = KitDataList;
export const DataListRow = KitDataListRow;

export function PortalSection({
  icon,
  ...props
}: Omit<ComponentProps<typeof KitPortalSection>, "icon"> & { icon?: string }) {
  return <KitPortalSection {...props} icon={icon ? <Icon name={icon} size={18} /> : undefined} />;
}

export function PortalActions(props: ComponentProps<typeof KitPortalActions>) {
  return (
    <KitPortalActions
      {...props}
      backIcon={props.back ? <Icon name="arrowBack" size={16} /> : undefined}
      primaryIcon={!props.primary.loading ? <Icon name="arrow_forward" size={16} /> : undefined}
    />
  );
}

export function SelectionCard({
  icon,
  ...props
}: Omit<ComponentProps<typeof KitSelectionCard>, "icon"> & { icon: string }) {
  return <KitSelectionCard {...props} icon={<Icon name={icon} size={18} />} />;
}

export function HubCard({
  icon,
  ...props
}: Omit<ComponentProps<typeof KitHubCard>, "icon" | "LinkComponent"> & { icon: string }) {
  return (
    <KitHubCard
      {...props}
      icon={<Icon name={icon} size={20} />}
      LinkComponent={Link as ComponentProps<typeof KitHubCard>["LinkComponent"]}
    />
  );
}

export function FormAlert({
  variant = "info",
  children,
}: {
  variant?: "info" | "error";
  children: ReactNode;
}) {
  return (
    <KitFormAlert
      variant={variant}
      icon={<Icon name={variant === "error" ? "error" : "verified"} size={18} />}
    >
      {children}
    </KitFormAlert>
  );
}
