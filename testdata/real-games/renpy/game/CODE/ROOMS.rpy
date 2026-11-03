## СТАРАЯ ВЕРСИЯ (остатки)

# текущая комната (по умолчанию кабинет)
default current_room = "kabinet"

init python:
    # with без паузы
    def trans(transition=Dissolve(.2)):
        renpy.transition(transition)
        renpy.restart_interaction()

    Trans = renpy.curry(trans)

    # добавить метку события, если такого события (label) ещё нет на карте сектора (x, y)
    def add_event(x, y, label, icon, xpos, ypos, available=True):
        global vevents

        if not label in vevents[y][x].keys(): vevents[y][x][label] = (icon, xpos, ypos, available)

    def off_event(x, y, label, available=False):
        global vevents

        if label in vevents[y][x].keys():
            icon, xpos, ypos, a = vevents[y][x][label]
            vevents[y][x][label] = (icon, xpos, ypos, available)

    # есть ли на карте невыполненные события
    def has_events(x, y):
        global vevents

        has = 0

        for i in vevents[y][x].values():
            icon, xpos, ypos, available = i

            if available: has += 1

        return has

    # есть ли на карте выполненные события
    def has_off_events(x, y):
        global vevents

        has = 0

        for i in vevents[y][x].values():
            icon, xpos, ypos, available = i

            if not available: has += 1

        return has

    # активировать сектор на карте
    def set_world_map(x, y, v=True):
        store.world_map[y][x] = v

# стандартное положение кнопок
define pnc_room_pos = {
    "up":    (.5,  .15),
    "down":  (.5,  .85),
    "left":  (.15, .5),
    "right": (.85, .5),
    "dot":   ( .5, .5),
}

# подсвечивание при наведении
transform pnc_hover_at(b=.15, t=.25):
    on show:
        alpha 0
        matrixcolor BrightnessMatrix(0)
        linear t alpha 1

    on hide:
        linear t alpha 0

    on idle:
        linear t matrixcolor BrightnessMatrix(0)

    on hover, selected:
        linear t matrixcolor BrightnessMatrix(b)

# плавно показать/скрыть
transform pnc_zoom_hover_at(t=.2, z=1.25, z0=1, a=1, a0=.75, b=.15):
    subpixel True

    on show:
        alpha 0 matrixcolor BrightnessMatrix(0)
        linear t alpha a0

    on hide:
        linear t alpha 0

    on hover, selected:
        ease t zoom z alpha a matrixcolor BrightnessMatrix(b)

    on idle:
        ease t zoom z0 alpha a0 matrixcolor BrightnessMatrix(0)

# подсвечивание при наведении на стол
transform pnc_desk_hover_at(t=.2, b=.2, a_idle=.05, a_hover=.5):
    on show:
        alpha 0 matrixcolor BrightnessMatrix(0)
        linear t alpha a_idle

    on hide:
        linear t alpha 0

    on idle:
        linear t alpha a_idle matrixcolor BrightnessMatrix(0)

    on hover, selected:
        linear t alpha a_hover matrixcolor BrightnessMatrix(b)

init python:
    # положение кликабельной кнопки на локации
    # если не указать « "dot" (pos=(111, 222)): », то будет центр экрана
    # для стрелок заданы в pnc_room_pos - тоже можно не указывать
    def pnc_get_pos(i):
        s = str(i.caption)
        x, y = pnc_room_pos.get(s, (.5, .5))
        x = i.kwargs.get("xpos", x)
        y = i.kwargs.get("ypos", y)

        return i.kwargs.get("pos", (x, y))

    # можно задать дополнительные трансформы,
    # кроме стандартного подсвечивания при наведении
    def pnc_get_at(i):
        s = str(i.caption)
        # обычное подсвечивание
        res = [ pnc_hover_at ]

        # для стола особое
        if s == "desk": res = [ pnc_desk_hover_at ]

        # для стрелок и точки - подсвечивание с зумом
        if s in [ str(i) for i in pnc_room_pos.keys() ]: res = [ pnc_zoom_hover_at ]

        at = i.kwargs.get("at", None )

        if at:
            if isinstance(at, tuple): at = list(at)

            if not isinstance(at, list): at = [ at ]

            res = res + at

        return res

# удалённые с локаций предметы
default pnc_deleted = [ ]

transform pnc_pass():
    pass

transform pnc_zoom(z=1):
    zoom z
