import React from 'react';

export const CircularProgress = ({
  progress = 0,
  stage = '',
  size = 26,
  strokeWidth = 2.5,
  onCancel,
  tooltip,
  cancelTooltip,
  onNavigateConsole,
}) => {
  const radius = (size - strokeWidth * 2) / 2;
  const circumference = 2 * Math.PI * radius;
  const clampedProgress = Math.min(100, Math.max(0, progress));
  const strokeDashoffset = circumference - (clampedProgress / 100) * circumference;
  const isIndeterminate = stage !== 'download' && stage !== 'done' && progress === 0;

  return (
    <div 
      className={`apple-circular-progress ${isIndeterminate ? 'indeterminate' : ''}`}
      onClick={(e) => {
        e.stopPropagation();
        if (onNavigateConsole) onNavigateConsole();
      }}
      title={tooltip || 'View console'}
      style={{ width: size, height: size }}
      role="button"
      tabIndex={0}
    >
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
        <circle
          className="progress-ring-bg"
          cx={size / 2}
          cy={size / 2}
          r={radius}
          strokeWidth={strokeWidth}
        />
        <circle
          className="progress-ring-fg"
          cx={size / 2}
          cy={size / 2}
          r={radius}
          strokeWidth={strokeWidth}
          strokeDasharray={circumference}
          strokeDashoffset={isIndeterminate ? circumference * 0.75 : strokeDashoffset}
        />
      </svg>
      <div 
        className="progress-ring-stop" 
        onClick={(e) => {
          e.stopPropagation();
          if (onCancel) onCancel();
        }}
        title={cancelTooltip || 'Cancel'}
      />
    </div>
  );
};

export default CircularProgress;
