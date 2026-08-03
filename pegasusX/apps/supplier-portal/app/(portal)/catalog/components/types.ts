export type SaleUnit = "UNIT" | "CASE";

export type CatalogProduct = {
  product_id: string;
  name: string;
  category_id: string;
  price_minor: number;
  currency: string;
  unit: string;
  unit_volume_vu: number;
  stackable?: boolean;
  max_stack_height?: number;
  units_per_case?: number;
  sale_unit?: SaleUnit;
  barcode?: string;
  image_url?: string;
  is_active: boolean;
  version: number;
};

export type CatalogCategory = {
  category_id: string;
  name: string;
};

export type CreateProductFormState = {
  name: string;
  category_id: string;
  description: string;
  price_minor: string;
  unit_volume_vu: string;
  units_per_case: string;
  sale_unit: SaleUnit;
  barcode: string;
};

export const ALLOWED_IMAGE_TYPES = ["image/jpeg", "image/png", "image/webp"];
export const MAX_IMAGE_SIZE = 5 * 1024 * 1024;
