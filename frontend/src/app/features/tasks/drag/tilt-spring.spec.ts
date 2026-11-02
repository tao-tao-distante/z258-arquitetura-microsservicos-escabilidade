import { leverFor, RESTING_TILT, stepTilt, TILT_CONFIG, TiltInput, TiltState } from './tilt-spring';

const frame = 1 / 60;

function run(state: TiltState, input: TiltInput, frames: number, visit?: (state: TiltState) => void): TiltState {
  let current = state;
  for (let index = 0; index < frames; index++) {
    current = stepTilt(current, input, frame);
    visit?.(current);
  }
  return current;
}

const still: TiltInput = { pointerVelocityX: 0, lever: 1, lifted: true };

describe('stepTilt', () => {
  it('settles back to rest when the pointer stops', () => {
    const settled = run({ ...RESTING_TILT, angle: 10 }, still, 180);

    expect(Math.abs(settled.angle)).toBeLessThan(0.05);
    expect(Math.abs(settled.angularVelocity)).toBeLessThan(0.5);
  });

  it('overshoots the target before settling, like a swinging card', () => {
    let lowest = Infinity;

    run({ ...RESTING_TILT, angle: 10 }, still, 180, (state) => (lowest = Math.min(lowest, state.angle)));

    expect(lowest).toBeLessThan(0);
  });

  it.each([
    { name: 'held above the center swings with the drag', lever: 1, velocity: 400, sign: 1 },
    { name: 'held above the center swings back when dragging left', lever: 1, velocity: -400, sign: -1 },
    { name: 'held below the center swings the opposite way', lever: -1, velocity: 400, sign: -1 },
  ])('$name', ({ lever, velocity, sign }) => {
    const swung = run(RESTING_TILT, { pointerVelocityX: velocity, lever, lifted: true }, 20);

    expect(Math.sign(swung.angle)).toBe(sign);
  });

  it('never exceeds the maximum angle', () => {
    let widest = 0;

    run(RESTING_TILT, { pointerVelocityX: 100_000, lever: 1, lifted: true }, 240, (state) => {
      widest = Math.max(widest, Math.abs(state.angle));
    });

    expect(widest).toBeLessThanOrEqual(TILT_CONFIG.maxAngle);
    expect(widest).toBeGreaterThan(TILT_CONFIG.maxAngle / 2);
  });

  it('clamps long frame gaps so the card never jumps', () => {
    const state = { ...RESTING_TILT, angle: 8 };

    expect(stepTilt(state, still, 5)).toEqual(stepTilt(state, still, TILT_CONFIG.maxStep));
  });

  it('lifts the card while held and lands it when released', () => {
    const lifted = run(RESTING_TILT, still, 60);
    const landed = run(lifted, { ...still, lifted: false }, 60);

    expect(lifted.scale).toBeCloseTo(TILT_CONFIG.liftScale, 3);
    expect(landed.scale).toBeCloseTo(1, 3);
  });
});

describe('leverFor', () => {
  it.each([
    { grabY: 0, height: 100, lever: 1 },
    { grabY: 100, height: 100, lever: -1 },
    { grabY: 50, height: 100, lever: 0.5 },
    { grabY: 75, height: 100, lever: -0.75 },
    { grabY: 500, height: 100, lever: -1 },
    { grabY: 10, height: 0, lever: 1 },
  ])('grabbing at $grabY of $height gives $lever', ({ grabY, height, lever }) => {
    expect(leverFor(grabY, height)).toBeCloseTo(lever, 5);
  });
});
