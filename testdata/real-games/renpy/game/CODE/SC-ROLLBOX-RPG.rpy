# дуга (пределы наклона в градусах) для всех пяти кнопок
define rb_w = 480
define rb_h = 100

# расстояние между кнопками, если roll == False
define rb_spacing = 10

# дуга на одну кнопку, если roll == True
define rb_angle = 10

# сила подсветки
define rb_brightness = .2

# вытягивание при наведении
define rb_xzoom = 1.025

# время подсвечивания
define rb_t = .25

# сколько кнопок сверху и снизу от выбранной кликабельны
define rb_clicable_count = 2

# плавность
define rb_k = 10. / config.framerate

# радиус круга для размещения кнопок
define rb_button_radius = 840

# отступ кнопок от центральной точки
define rb_xoffset = 300

# стиль кнопок
style rb_button is button:
    xysize (rb_w, rb_h)
    background Frame("button idle", 5, 5)
    hover_background Frame("button hover", 5, 5)
    insensitive_background Frame("button insensitive", 5, 5)
    selected_foreground Frame("button selected", 5, 5)

# стиль текста кнопок
style rb_button_text is text:
    align (.5, .5)
    layout "subtitle"
    text_align .5
    insensitive_color "#fff8"
    color "#fff"
    size 28

# стиль мелкого текста кнопок
style rb_button_text_small is rb_button_text:
    color "#fe8"
    insensitive_color "#fe8b"
    size 20

# плавное появление/исчезание
transform rb_show_hide(t=rb_t):
    on show:
        alpha 0
        linear t alpha 1

    on hide:
        linear t alpha 0

# кнопка мыши нажата на области списка
define rb_mouse_down = False

# кнопка мыши наведена на область списка
define rb_mouse_hover = False

init python:
    import math

    # если это не список, то сделать единственным элементом списка
    def rb_make_list(param, none=[ ]):
        if param is None:
            return none

        if not isinstance(param, list):
            param = [ param ]

        return param


    # нормализовать список разрешённых кнопок
    def rb_normalize_allow(buttons, allow):
        if allow:
            return [i for i in allow if 0 <= i < len(buttons)]

        return list(range(len(buttons)))


    # привести rb_current к ближайшей разрешённой кнопке
    def rb_fix_current(buttons, allow):
        global rb_current

        if not buttons:
            rb_current = 0
            return 0

        allow = rb_normalize_allow(buttons, allow)

        if not allow:
            rb_current = 0
            return 0

        if rb_current in allow:
            return rb_current

        # ищем ближайшую доступную кнопку после текущей
        for i in allow:
            if i >= rb_current:
                rb_current = i
                return rb_current

        # если после нет — берём последнюю доступную
        rb_current = allow[-1]
        return rb_current


    # позиция кнопки внутри списка разрешённых
    def rb_allow_pos(i, allow):
        if not allow:
            return i

        if i in allow:
            return allow.index(i)

        return 0


    # динамический трансформ для прокрутки
    def rb_at_f(index, buttons, x, y, roll, allow, trans, st, at):
        global rb_mouse_pos, rb_current

        trans.subpixel = True

        if isinstance(x, float):
            x *= config.screen_width

        if isinstance(y, float):
            y *= config.screen_height

        allow = rb_normalize_allow(buttons, allow)

        if not allow:
            trans.alpha = 0
            return 1. / config.framerate

        rb_fix_current(buttons, allow)

        if renpy.variant("touch"):
            if rb_mouse_pos is not None:
                x1, y1 = rb_mouse_pos
                x2, y2 = renpy.get_mouse_pos()

                if math.hypot(x2 - x1, y2 - y1) > rb_h / 2:
                    if y2 < y1:
                        rb_next(buttons, allow)

                    elif y2 > y1:
                        rb_prev(buttons, allow)

                    rb_mouse_pos = renpy.get_mouse_pos()

        current = rb_allow_pos(rb_current, allow)
        index = rb_allow_pos(index, allow)

        if roll:
            trans.rotate = 0 if trans.rotate is None else trans.rotate

            angle = (index - current) * rb_angle
            trans.rotate += (angle - trans.rotate) * rb_k

            a = math.radians(trans.rotate + 90)

            x += rb_button_radius * math.sin(a)
            y -= rb_button_radius * math.cos(a)

            x, y = int(x), int(y)

            trans.anchor = (0, .5)
            trans.xpos = x - rb_button_radius + rb_xoffset
            trans.ypos = y

            r = trans.rotate % 360
            r = abs(360 - r) if r > 180 else r

            arc = (rb_clicable_count + .5) * rb_angle
            k = max(0, min(1, 1 - r / arc))

        else:
            trans.yoffset = 0 if trans.yoffset is None else trans.yoffset

            yoffset = (index - current) * (rb_h + rb_spacing)
            trans.yoffset += (yoffset - trans.yoffset) * rb_k

            trans.xpos = int(x)
            trans.ypos = int(y)
            trans.anchor = (.5, .5)

            yo = (rb_clicable_count + .5) * (rb_h + rb_spacing)
            k = max(0, min(1, 1 - abs(trans.yoffset / yo)))

        trans.zoom = .75 + .25 * k
        trans.alpha = k

        return 1. / config.framerate


    # скролл вниз
    def rb_next(buttons, allow):
        global rb_current

        allow = rb_normalize_allow(buttons, allow)

        if not allow:
            rb_current = 0
            return

        rb_fix_current(buttons, allow)

        pos = allow.index(rb_current)

        if pos < len(allow) - 1:
            rb_current = allow[pos + 1]

        renpy.set_focus("rollbox", "rb_%s" % (rb_current))


    RBNext = renpy.curry(rb_next)


    # скролл вверх
    def rb_prev(buttons, allow):
        global rb_current

        allow = rb_normalize_allow(buttons, allow)

        if not allow:
            rb_current = 0
            return

        rb_fix_current(buttons, allow)

        pos = allow.index(rb_current)

        if pos > 0:
            rb_current = allow[pos - 1]

        renpy.set_focus("rollbox", "rb_%s" % (rb_current))


    RBPrev = renpy.curry(rb_prev)

# трансформ для прокрутки
transform rb_at(index, buttons, x, y, roll=True, allow=None, t=rb_t, b=rb_brightness, z=rb_xzoom):
    subpixel True

    # наведение
    parallel:
        on idle:
            linear t matrixcolor BrightnessMatrix(0) xzoom 1

        on hover:
            linear t matrixcolor BrightnessMatrix(0 if index == rb_current else b) xzoom z

    # прокрутка
    parallel:
        function renpy.curry(rb_at_f)(index, buttons, x, y, roll, allow)

# по умолчанию выбрана первая кнопка списка
default rb_current = 0

# для скролла
define rb_current_last = None

# наведение на кнопку, свайп-зону или никуда
define rb_hovered = None

# позиция опускания кнопки мыши
define rb_mouse_pos = None

# действие нажатой кнопки
define rb_action = None

init python:
    # нажать кнопку мыши
    def rb_mouse_down():
        global rb_current_last, rb_mouse_pos

        rb_current_last = rb_current
        rb_mouse_pos = renpy.get_mouse_pos()

    RBMouseDown = renpy.curry(rb_mouse_down)

    # отпустить кнопку мыши
    def rb_mouse_up(buttons):
        global rb_hovered, rb_mouse_pos, rb_action

        # если отпускаем кнопку
        if rb_mouse_pos is None:
            rb_mouse_pos = renpy.get_mouse_pos()

        x1, y1 = rb_mouse_pos
        x2, y2 = renpy.get_mouse_pos()

        # если не было скролла
        if rb_current_last == rb_current:
            # считаем, что это клик
            if isinstance(rb_hovered, int):
                rb_action = rb_hovered
                renpy.restart_interaction()

        rb_mouse_pos = None

    RBMouseUp = renpy.curry(rb_mouse_up)

# стиль считывателя свайпов
style rb_frame is empty:
    anchor (0, .5)
    xysize (rb_xoffset + int(rb_w * rb_xzoom * 1.1), config.screen_height)

# прокручивающийся по дуге список
screen rollbox(buttons, x=.25, y=.5, roll=True, allow=None, back=False):
    $ btns = list(buttons)[:]

    if back and not ( _("НАЗАД"), "back", 0, [ ] ) in btns:
        $ btns = btns + [ ( _("НАЗАД"), "back", 0, [ ] ) ]

    # видимые кнопки (если не указывать, то все)
    $ allo = list(allow)[:] if allow else list(range(len(btns)))

    $ allo = list([ i for i in allo if 0 <= i < len(btns) ])

    if back:
        $ allo = allo + [ len(btns) - 1 ]

    $ rb_fix_current(btns, allo)

    # сброс переменных для свайпа
    on [ "show", "hide" ] action SetVariable("rb_hovered", None), SetVariable("rb_mouse_pos", None), SetVariable("rb_action", None), SetVariable("rb_current_last", None)

    key [ "rollback", "viewport_uparrow" ] action RBPrev(btns, allo)
    key [ "rollforward", "viewport_downarrow" ] action RBNext(btns, allo)
    key [ "game_menu" ] action NullAction()

    style_prefix "rb"

    button:
        style "rb_frame"
        focus_mask True
        pos (x, y)
        hovered SetVariable("rb_hovered", "rollbox")
        unhovered SetVariable("rb_hovered", None)
        action NullAction()

    fixed:
        xfill True
        yfill True
        at rb_show_hide

        $ index = 0

        # веер кнопок
        for i in range(len(btns)):
            $ btn_t = btns[i]
            $ caption, name, ap, temp = btn_t

            if i in allo:
                button:
                    style "b_button"
                    focus_mask True
                    at rb_at(i, btns, x, y, roll, allo)
                    sensitive (abs(index - rb_allow_pos(rb_current, allo)) <= rb_clicable_count)
                    unhovered SetVariable("rb_hovered", "rollbox")
                    hovered SetVariable("rb_hovered", i)
                    action ( NullAction() if renpy.variant("touch") else Return(i) )

                    if not roll:
                        sensitive b_who.ap >= ap

                    # стопка строк
                    vbox style "vbox" align (.5, .5):
                        if roll:
                            text caption style "rb_button_text"

                        else:
                            for t in range(2):
                                if t:
                                    # вторая строка в кнопке
                                    if ap > 0:
                                        $ caption = str(ap)

                                    else:
                                        $ caption = ""

                                # если кнопка не пустая, покажем её
                                if caption:
                                    hbox:
                                        xalign .5

                                        if t:
                                            text _("ОД: ") style "rb_button_text_small"

                                        text caption:
                                            # первая строка отличается от прочих
                                            if t:
                                                style "rb_button_text_small"

                                            else:
                                                style "rb_button_text"

                $ index += 1

    if renpy.variant("touch"):
        key "mousedown_1" action RBMouseDown()
        key "mouseup_1" action RBMouseUp(btns)

        if rb_action is not None:
            timer .0001 action Return(rb_action)

# прокручивающийся список
screen scrollbox(buttons, x=.25, y=.5, roll=False, allow=None, back=False):
    use rollbox(buttons, x, y, roll, allow, back)
