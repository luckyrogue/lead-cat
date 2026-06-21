export function canManageMeeting(
  meeting: { organizer: string } | undefined,
  user: { email: string; role: string } | null | undefined
): boolean {
  if (!meeting || !user) {
    return false
  }
  return meeting.organizer === user.email || user.role === "owner"
}
