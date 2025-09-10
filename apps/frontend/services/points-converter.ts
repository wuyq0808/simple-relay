/**
 * Points conversion utility for converting internal points to display points.
 * 
 * Internal storage: points = cost * 1,000,000
 * Display to users: display points = cost * 100 (or points / 10,000)
 */

/**
 * Convert points (internal storage) to display points (user-facing)
 * @param points - Points stored internally as cost * 1,000,000
 * @returns Display points as cost * 100
 */
export function pointsToDisplayPoints(points: number): number {
  return Math.round(points / 10000);
}