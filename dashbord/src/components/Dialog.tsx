import type { JSX } from "preact";
import { useState } from "preact/hooks";

import { navigate, url } from "../lib/router.tsx";

type DialogProps = {
  id: string;
  children: preact.ComponentChildren;
} & JSX.HTMLAttributes<HTMLDialogElement>;

export const Dialog = ({
  id,
  onClick,
  onClose,
  ...props
}: DialogProps) => {
  const [dialogElem, setRef] = useState<HTMLDialogElement | null>(null);

  const isOpen = url.params.dialog === id;
  if (dialogElem) {
    if (!isOpen && dialogElem.open) {
      dialogElem.close();
    } else if (isOpen && !dialogElem.open) {
      dialogElem.showModal();
    }
  }

  return (
    <dialog
      {...props}
      id={id}
      onClick={(event) => {
        dialogElem === event.target && dialogElem?.close();
        typeof onClick === "function" && onClick(event);
      }}
      onClose={(event) => {
        typeof onClose === "function" && onClose(event);
        if (!isOpen) return;
        navigate({ params: { dialog: null } });
      }}
      ref={setRef}
    >
    </dialog>
  );
};

export const DialogModal = ({ children, ...props }: DialogProps) => {
  return (
    <Dialog class="modal" {...props}>
      <div class="modal-box w-auto">
        <form method="dialog">
          <button
            type="submit"
            class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
          >
            ✕
          </button>
        </form>
        {children}
      </div>
    </Dialog>
  );
};
