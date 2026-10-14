import React from 'react';
import { CheckCircle2, AlertCircle, Info } from 'lucide-react';

export const ToastNotification = ({ toast }) => {
  if (!toast) return null;

  return (
    <div className="toast-container">
      <div className={`toast-pill ${toast.type || 'info'}`}>
        {toast.type === 'success' && <CheckCircle2 size={15} color="var(--system-green)" />}
        {toast.type === 'error' && <AlertCircle size={15} color="var(--system-red)" />}
        {toast.type === 'info' && <Info size={15} color="var(--system-blue)" />}
        <span>{toast.message}</span>
      </div>
    </div>
  );
};

export default ToastNotification;
