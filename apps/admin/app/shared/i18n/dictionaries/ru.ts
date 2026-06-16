import type { Dict } from "~/shared/i18n/types"

export const ru: Dict = {
  common: {
    save: "Сохранить",
    cancel: "Отмена",
    delete: "Удалить",
    edit: "Редактировать",
    loading: "Загрузка…",
    retry: "Повторить",
    back: "Назад",
    saved: "Сохранено",
    saveFailed: "Не удалось сохранить",
    failedToLoad: "Не удалось загрузить.",
  },
  nav: {
    dashboard: "Главная",
    members: "Участники",
    invites: "Приглашения",
    meetings: "Встречи",
    settings: "Настройки",
  },
  dashboard: {
    eyebrow: "Lead Cat Admin",
    welcome: "Добро пожаловать, {name}",
    descriptionOrg: "Краткий обзор {org}.",
    descriptionDefault: "Краткий обзор вашего рабочего пространства.",
    statOrganization: "Организация",
    statOrganizationHintNone: "Организация не выбрана",
    statOrganizationHintSlug: "Слаг: {slug}",
    statMembers: "Участники",
    statMembersHint: "Пользователи с доступом к организации",
    statInvites: "Ожидающие приглашения",
    statInvitesHint: "Приглашения, ожидающие принятия",
  },
  sidebar: {
    selectOrganization: "Выбрать организацию",
    organizationLabel: "Организация",
    organizationsMenuLabel: "Организации",
    logoutLabel: "Выйти",
    loadingWorkspace: "Загрузка рабочего пространства…",
  },
}
