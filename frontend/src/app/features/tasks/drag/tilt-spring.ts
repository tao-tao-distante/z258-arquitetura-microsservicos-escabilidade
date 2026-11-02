export interface TiltConfig {
  stiffness: number;
  dampingRatio: number;
  sensitivity: number;
  maxAngle: number;
  maxStep: number;
  velocitySmoothing: number;
  liftScale: number;
  liftRate: number;
}

export interface TiltState {
  angle: number;
  angularVelocity: number;
  velocityX: number;
  scale: number;
}

export interface TiltInput {
  pointerVelocityX: number;
  lever: number;
  lifted: boolean;
}

export const TILT_CONFIG: TiltConfig = {
  stiffness: 180,
  dampingRatio: 0.6,
  sensitivity: 0.03,
  maxAngle: 14,
  maxStep: 0.032,
  velocitySmoothing: 0.3,
  liftScale: 1.04,
  liftRate: 18,
};

export const RESTING_TILT: TiltState = { angle: 0, angularVelocity: 0, velocityX: 0, scale: 1 };

function clamp(value: number, limit: number): number {
  return Math.min(limit, Math.max(-limit, value));
}

export function stepTilt(state: TiltState, input: TiltInput, dtSeconds: number, config = TILT_CONFIG): TiltState {
  const dt = Math.min(Math.max(dtSeconds, 0), config.maxStep);
  const velocityX = state.velocityX + (input.pointerVelocityX - state.velocityX) * config.velocitySmoothing;
  const target = clamp(velocityX * config.sensitivity * input.lever, config.maxAngle);
  const damping = 2 * config.dampingRatio * Math.sqrt(config.stiffness);
  const acceleration = -config.stiffness * (state.angle - target) - damping * state.angularVelocity;
  let angularVelocity = state.angularVelocity + acceleration * dt;
  let angle = state.angle + angularVelocity * dt;
  if (Math.abs(angle) > config.maxAngle) {
    angle = clamp(angle, config.maxAngle);
    angularVelocity = 0;
  }
  const scaleTarget = input.lifted ? config.liftScale : 1;
  const scale = state.scale + (scaleTarget - state.scale) * (1 - Math.exp(-config.liftRate * dt));
  return { angle, angularVelocity, velocityX, scale };
}

export function leverFor(grabY: number, height: number): number {
  if (height <= 0) {
    return 1;
  }
  const half = height / 2;
  const offset = clamp((half - grabY) / half, 1);
  const sign = offset >= 0 ? 1 : -1;
  return sign * (0.5 + 0.5 * Math.abs(offset));
}
