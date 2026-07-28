import { ApiClient, ApiError } from "@pegasusx/api-client";
import type {
  HomeNodeType,
  Role,
  SupplierOrgMemberCreateRequest,
  SupplierTopologyResponse,
} from "@pegasusx/types";

export function supplierScopeId(): string {
  if (typeof window === "undefined") {
    return "supplier";
  }
  return window.localStorage.getItem("supplier_id")?.trim() || "supplier";
}

export type ReadyState = {
  status: "ready";
  topology: SupplierTopologyResponse;
  orgMembers: Awaited<ReturnType<ApiClient["getSupplierOrgMembers"]>>["items"];
  drivers: Awaited<ReturnType<ApiClient["getSupplierFleetDrivers"]>>["items"];
  vehicles: Awaited<ReturnType<ApiClient["getSupplierFleetVehicles"]>>["items"];
};

export type OrgFormState = {
  name: string;
  email: string;
  phone: string;
  password: string;
  role: Role;
  nodeType: HomeNodeType;
  nodeID: string;
};

export type DriverFormState = {
  name: string;
  phone: string;
  pin: string;
  homeNodeType: HomeNodeType;
  homeNodeID: string;
  vehicleID: string;
};

export type VehicleFormState = {
  label: string;
  licensePlate: string;
  homeNodeType: HomeNodeType;
  homeNodeID: string;
};

export const orgRoleOptions: Array<{ value: Role; label: string }> = [
  { value: "ADMIN", label: "Supplier operator" },
  { value: "WAREHOUSE_ADMIN", label: "Warehouse admin" },
  { value: "FACTORY_ADMIN", label: "Factory admin" },
  { value: "PAYLOAD", label: "Payload staff" },
];

export const defaultOrgForm: OrgFormState = {
  name: "",
  email: "",
  phone: "",
  password: "",
  role: "WAREHOUSE_ADMIN",
  nodeType: "WAREHOUSE",
  nodeID: "",
};

export const defaultDriverForm: DriverFormState = {
  name: "",
  phone: "",
  pin: "",
  homeNodeType: "WAREHOUSE",
  homeNodeID: "",
  vehicleID: "",
};

export const defaultVehicleForm: VehicleFormState = {
  label: "",
  licensePlate: "",
  homeNodeType: "WAREHOUSE",
  homeNodeID: "",
};

export function StatusText(props: { message: string; isError: boolean }) {
  return (
    <p
      className="md-typescale-body-medium"
      role={props.isError ? "alert" : undefined}
      style={{ color: props.isError ? "var(--color-md-error)" : "var(--color-md-outline)" }}
    >
      {props.message}
    </p>
  );
}

export function buildOrgMemberRequest(form: OrgFormState): SupplierOrgMemberCreateRequest {
  const request: SupplierOrgMemberCreateRequest = {
    name: form.name,
    email: form.email || undefined,
    phone: form.phone,
    password: form.password,
    supplier_role: form.role,
  };
  if (form.role === "WAREHOUSE_ADMIN") {
    request.assigned_warehouse_id = form.nodeID;
  }
  if (form.role === "FACTORY_ADMIN") {
    request.assigned_factory_id = form.nodeID;
  }
  if (form.role === "PAYLOAD") {
    if (form.nodeType === "WAREHOUSE") {
      request.assigned_warehouse_id = form.nodeID;
    } else {
      request.assigned_factory_id = form.nodeID;
    }
  }
  return request;
}

export function orgEffectiveNodeType(form: OrgFormState): HomeNodeType {
  if (form.role === "FACTORY_ADMIN") {
    return "FACTORY";
  }
  return form.nodeType;
}

export function nodeOptionsFor(nodeType: HomeNodeType, topology: SupplierTopologyResponse) {
  if (nodeType === "FACTORY") {
    return topology.factories.map((factory) => ({ value: factory.factory_id, label: factory.name }));
  }
  return topology.warehouses.map((warehouse) => ({ value: warehouse.warehouse_id, label: warehouse.name }));
}

export function describeMemberNode(
  member: ReadyState["orgMembers"][number],
  topology: SupplierTopologyResponse,
) {
  if (member.assigned_warehouse_id) {
    return describeHomeNode("WAREHOUSE", member.assigned_warehouse_id, topology);
  }
  if (member.assigned_factory_id) {
    return describeHomeNode("FACTORY", member.assigned_factory_id, topology);
  }
  return "Supplier-wide";
}

export function describeHomeNode(nodeType: HomeNodeType, nodeID: string, topology: SupplierTopologyResponse) {
  if (nodeType === "FACTORY") {
    const factory = topology.factories.find((item) => item.factory_id === nodeID);
    return factory ? `Factory · ${factory.name}` : nodeID;
  }
  const warehouse = topology.warehouses.find((item) => item.warehouse_id === nodeID);
  return warehouse ? `Warehouse · ${warehouse.name}` : nodeID;
}

export function describeVehicle(vehicleID: string | undefined, vehicles: ReadyState["vehicles"]) {
  if (!vehicleID) {
    return "Assigned later";
  }
  const vehicle = vehicles.find((item) => item.vehicle_id === vehicleID);
  if (!vehicle) {
    return vehicleID;
  }
  return vehicle.label ? `${vehicle.label} · ${vehicle.license_plate}` : vehicle.license_plate;
}

export function formatRole(role: Role) {
  switch (role) {
    case "WAREHOUSE_ADMIN":
      return "Warehouse admin";
    case "FACTORY_ADMIN":
      return "Factory admin";
    case "PAYLOAD":
      return "Payload staff";
    case "ADMIN":
      return "Supplier operator";
    default:
      return role;
  }
}

export function toErrorMessage(error: unknown) {
  if (error instanceof ApiError) {
    return error.message;
  }
  if (error instanceof Error && error.message.trim() !== "") {
    return error.message;
  }
  return "Request failed.";
}

export function isErrorMessage(message: string) {
  const lower = message.toLowerCase();
  return lower.includes("failed") || lower.includes("invalid") || lower.includes("exists");
}

export function newIdempotencyKey() {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `pegasusx-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}
