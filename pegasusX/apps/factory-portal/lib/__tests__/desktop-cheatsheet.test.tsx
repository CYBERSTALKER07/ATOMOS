import { describe, it, expect, vi } from "vitest";
import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import { DesktopKeyboardCheatsheet } from "@/components/ui/DesktopKeyboardCheatsheet";

describe("Factory Portal Desktop KeyboardCheatsheet Component", () => {
  it("does not render when open is false", () => {
    const { container } = render(
      <DesktopKeyboardCheatsheet open={false} onClose={vi.fn()} />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("renders factory keyboard shortcuts dialog when open is true", () => {
    render(
      <DesktopKeyboardCheatsheet open={true} onClose={vi.fn()} />,
    );
    expect(screen.getByText("Desktop Keyboard Shortcuts")).toBeDefined();
    expect(screen.getByText("Global Navigation")).toBeDefined();
    expect(screen.getByText("Production & Batch Planning")).toBeDefined();
    expect(screen.getByText("Quality Assurance & Contamination")).toBeDefined();
    expect(screen.getByText("BOM Recipe & Production Batch Planning")).toBeDefined();
  });

  it("triggers onClose when clicking the close button or pressing Escape", () => {
    const onClose = vi.fn();
    render(<DesktopKeyboardCheatsheet open={true} onClose={onClose} />);
    
    const closeBtn = screen.getByLabelText("Close shortcuts");
    fireEvent.click(closeBtn);
    expect(onClose).toHaveBeenCalledTimes(1);

    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(2);
  });
});
