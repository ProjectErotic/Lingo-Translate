# без этого не пускают в стим
define config.default_fullscreen = True

# имена в книгах жирным
define who_bold = True

# режим игры (home, book)
default mode = ""

# время на появление картинок
define show_t = .5

# отступ стрипов от нижнего края
define pad_bottom = 20

# нужна ли подсветка
default persistent.text_light = False

# для quick_menu
image quick_check = DynamicDisplayable(quick_check_f)

# пустой пиксель
image null = Null(1, 1)

# пустой пиксель
image bg null = Null(1, 1)

style button:
    activate_sound "audio/click.ogg"

style image_button:
    activate_sound "audio/click.ogg"

style choice_button:
    activate_sound "audio/click.ogg"

image ctc_blink:
    "ctc"
    align (.5, .5)
    yoffset 5
    alpha 0
    .25
    linear 1 alpha 1
    .25
    linear 1 alpha 0
    repeat

# горизонтальная трясучка
transform hshake(count=6, d=8, dt=.25):
    subpixel True
    xoffset 0
    easeout dt * .25 xoffset d
    ease dt * .25 xoffset -d
    easein dt * .25 xoffset 0
    repeat count

# горизонтальная трясучка
transform vshake(count=6, d=8, dt=.25):
    subpixel True
    yoffset 0
    easeout dt * .25 yoffset d
    ease dt * .25 yoffset -d
    easein dt * .25 yoffset 0
    repeat count

define shake = hshake

# трансформы для текстбокса
# define config.window_show_transition = None
# define config.window_hide_transition = None

# transform say_show_hide(t=.35, xz=.5):
    # subpixel True
    # xzoom xz alpha 0

    # on show:
        # easein t xzoom 1 alpha 1

    # on hide:
        # easeout t xzoom xz alpha 0

# трансформы для неймбокса
transform who_at(who):
    function renpy.curry(who_at_f)(who)

transform who_at_showed():
    xzoom 1
    yzoom 1
    alpha 1

transform who_at_hided():
    xzoom 0
    yzoom 1
    alpha 0

transform who_at_show(t=.25):
    subpixel True
    xzoom 0
    yzoom 1
    alpha 0
    ease t alpha 1 xzoom 1
    function renpy.curry(who_set_f)("who_at_showed")

transform who_at_hide(t=.25):
    subpixel True
    xzoom 0
    yzoom 1
    alpha 0
    function renpy.curry(who_set_f)("who_at_hided")

default last_say_who = None
default who_at_s = "who_at_hided"

# рекламный баннер
image link1 = Ani("link1 ", 120, .05, None)
image link2 = Ani("link2 ", 38, .05, None)

init -2 python:
    def has_image(name):
        return name in renpy.list_images()

    # аналог Animation (для сокращения писанины и с добавлением переходов)
    def Ani(name, count=6, t=.05, effect=Dissolve(.05), first=1, loop=True, reverse=False, **kwargs):
        imgs = [ name + str(i + first) for i in range(count) ]

        if reverse: imgs = list(reversed(imgs))

        args = sum([ [ imgs[i], t, effect ] for i in range(count) ], [ ])[:(None if loop else -2)]

        return Transform(anim.TransitionAnimation(*args), **kwargs)

init python:
    # мини-менеджер событий, замена переменным
    # можно использовать и для журнала заданий
    class E():
        def __init__(self):
            self.events = [ ]
            self.dones = [ ]

        # создать одно или несколько событий e.add("sleep", "eat", ...)
        def add(self, *args):
            for i in args:
                if not i in self.events: self.events.append(i)

        # удалить одно или несколько событий e.res("sleep", "eat", ...)
        def res(self, *args):
            for i in args:
                if i in self.events: self.events.remove(i)

        # пометить событие выполненным, даже если оно не существует
        def done(self, *args):
            for i in args:
                if not i in self.dones: self.dones.append(i)

        # отменить пометку выполнено
        def undone(self, *args):
            for i in args:
                if i in self.dones: self.dones.remove(i)

        # все ли указанные задания выполнены (могут уже быть удалены)
        def is_done(*args):
            for i in args:
                if not i in self.dones: return False

            return True

        # хотя бы одно задание выполнено
        def one_done(self, *args):
            for i in args:
                if i in self.dones: return True

            return False

        # есть все события из списка через запятую e.has("sleep", "eat", ...)
        def has(self, *args):
            for i in args:
                if not i in self.events: return False

            return True

        # есть хотя бы одно событие из списка через запятую e.one("sleep", "eat", ...)
        def one(self, *args):
            for i in args:
                if i in self.events: return True

            return False

        # список всех задач с пометкой о выполнении
        def all(self):
            res = [ (i, ( i in self.dones ) ) for i in self.events ]

            if not res: res = [ (_("Задачи отсутствуют"), False) ]

            return res

# мини-менеджер событий, замена переменным
default e = E()

# журнал заданий
default tasks = E()

# по умолчанию вместо имени гг - вампир
default hero_name = _("Вампир")
default girl_name = _("Лили")




init python:
    # получить имя персонажа character
    def get_who_name(character):
        if character is None: return None

        ch = globals().get(character, "")

        if not ch: return character

        return ch.name

    # смена трансформа для неймбокса
    def who_set_f(s, trans, st, at):
        store.who_at_s = s
        store.last_say_who = _last_say_who
        renpy.restart_interaction()

    # меняем трансформы в зависимости от того, нужно ли показывать неймбокс
    def who_at_f(who, trans, st, at):
        global who_at_s

        n_new, n_old = get_who_name(who), get_who_name(last_say_who)

        # если персонаж сменился и видно текстбокс
        if (n_new != n_old) and renpy.get_screen("say"):
            # если имя прячется
            if (not n_old is None) and (n_new is None):
                if not who_at_s in [ "who_at_hide", "who_at_hided" ]:
                    who_at_s = "who_at_hide"
                    renpy.restart_interaction()

            # если новое имя появляется
            elif (not n_new is None) and (n_old is None):
                if not who_at_s in [ "who_at_show", "who_at_showed" ]:
                    who_at_s = "who_at_show"
                    renpy.restart_interaction()

        return 1. / config.framerate

    # для on "hide" в screen say
    def say_hide():
        if get_who_name(_last_say_who):
            store.who_at_s = "who_at_hided"
            store.last_say_who = None

    SayHide = renpy.curry(say_hide)

    # применить эффект к спрайту
    def do_shake(tag, at_list=hshake):
        renpy.show(tag, at_list=make_list(at_list))
        renpy.restart_interaction()

    # у спрайтов из этих папок вместо пробелов будут нижние_минусы - все_из_одного_тега
    layered_prefixes = [ "ch" ]

    # следим, чтобы полоса с кнопками quick_menu была показана в соотвествии с переменной
    def quick_check_f(st, at):
        if persistent.quick_menu and not has_screen("quick_line"):
            if not _windows_hidden: Show("quick_line")()

        elif has_screen("quick_line") and not persistent.quick_menu:
            Hide("quick_line")()

        return "null", 1. / config.framerate

    # показан ли экран (или один из экранов)
    def has_screen(*args, **kwarg):
        layer = kwarg.get("layer", None)

        for i in make_list(args):
            if renpy.get_screen(i, layer): return True

        return False

    # открыто ли game menu
    def game_menu():
        return renpy.context()._menu or renpy.get_screen("confirm")

    # смешиваем канал прозрачности у двух цветов
    def ch_alpha(color, alpha="#0003"):
        return Color(rgb=Color(color).rgb, alpha=Color(alpha).alpha)

    # показать говорящий спрайт, пока на экран выводится его реплика
    # и показать молчащий спрайт, когда реплика полностью появится в текстбоксе
    def l2d_show(tag="gg", end="talk"):
        layer = "master"

        if tag in renpy.get_showing_tags(layer):
            attrs = list(renpy.get_attributes(tag, layer) or [])

            mouth_tags = [ "talk", "mute", "open_mouth", "close_mouth" ]

            # Если нужный тег уже стоит И других тегов рта нет — ничего не делаем.
            # Это защищает от двойного запуска одной и той же анимации.
            if end in attrs:
                has_other_mouth = False

                for mouth in mouth_tags:
                    if mouth != end and mouth in attrs:
                        has_other_mouth = True
                        break

                if not has_other_mouth:
                    return

            # Убираем ВСЕ теги рта из любого места строки.
            # Например: gg anger talk casual_blood_dress mute
            # станет:   gg anger casual_blood_dress
            attrs = [ a for a in attrs if a not in mouth_tags ]

            # Добавляем только нужную анимацию.
            attrs.append(end)

            spr = " ".join([ tag ] + attrs)

            renpy.show(spr, what=spr, layer=layer)
            renpy.restart_interaction()


    # по окончании диалога, запоминаем последнего говорившего
    def normalize_who_name(name):
        if not name:
            return None

        aliases = {
            "hero_m": "hero",
            "girl_m": "girl",
            "vv_m": "vv",
            "ss_m": "ss",
            "gg_m": "gg",
        }

        return aliases.get(name, name)


    def ch_callback(event, interact=True, **kwargs):
        if not interact:
            return

        if event == "begin":
            new_who = kwargs.get("who", None)

            if new_who is None:
                new_who = _last_say_who

            new_name = get_who_name(new_who)
            old_name = get_who_name(last_say_who)

            # Превращаем hero_m в hero, girl_m в girl и т.д.
            new_key = normalize_who_name(new_name)
            old_key = normalize_who_name(old_name)

            # Если реплика без говорящего:
            # namebox вообще не создаётся в screen say, поэтому ничего не анимируем.
            if not new_key:
                store.who_at_s = "who_at_hided"
                return

            # Если это тот же персонаж, даже если hero -> hero_m:
            # не запускаем анимацию заново.
            if old_key and new_key == old_key:
                store.who_at_s = "who_at_showed"
                return

            # Если реально сменился персонаж — показываем имя и подложку.
            store.who_at_s = "who_at_show"

        if event == "end":
            current_name = get_who_name(_last_say_who)
            current_key = normalize_who_name(current_name)

            if current_key:
                store.last_say_who = _last_say_who
            else:
                store.last_say_who = None

    config.character_callback = ch_callback

    # свой callback для говорящего live2d спрайта
default talk_anim_id = 0

init python:
    def talk_callback(event, interact=True, tag="gg", **kwargs):
        if not interact:
            return

        if not tag:
            return

        if event == "show_done":
            # Ничего не запускаем.
            # talk ты сам уже запустил через:
            # show gg anger talk casual_blood_dress
            pass

        elif event == "slow_done":
            store.talk_anim_id += 1
            current_id = store.talk_anim_id
            renpy.invoke_in_thread(stop_talk_after_delay, tag, current_id)


    def stop_talk_after_delay(tag, anim_id):
        import time
        time.sleep(0.7)

        if anim_id == store.talk_anim_id:
            renpy.invoke_in_main_thread(l2d_show, tag, "mute")

    # добавляем параметры по умолчанию всем персонажам
    def Ch(name=None, kind=None, **kwargs):
        color = kwargs.pop("color", kwargs.get("who_color", "#a22"))
        kwargs["who_color"] = color

        # side image по умолчанию - пустой ромбик
        kwargs["image"] = kwargs.get("image", "none")

        # картинка для click-to-continue
        kwargs["ctc"] = kwargs.get("ctc", "ctc_blink")
        kwargs["ctc_position"] = kwargs.get("ctc_position", "nestled")

        if name: kwargs["what_prefix"] = kwargs.get("what_prefix", "- ")

        # добавляем свечение того же цвета, что и текст/имя
        if persistent.text_light:
            for key, default in ( ("who", "#a22"), ("what", "#fff") ):
                color = kwargs.get(key + "_color", default)
                out = [ (0, ch_alpha(color, "#0003"), 4, 1), (0, ch_alpha(color, "#0005"), 3, 1), (0, "#000", 2, 1) ]
                kwargs[key + "_outlines"] = kwargs.get(key + "_outlines", out)

        return Character(name, kind, **kwargs)

    narrator = Ch(None, image=None)

    dd = Ch(None, image=None, what_style="centered_text", window_style="centered_window", statement_name="say-centered", what_outlines=[ (0, "#fff", 1, 0) ], what_size=gui.text_size+8, what_font="font/pen.ttf", what_suffix="\n", what_line_spacing=4)
    ddd = Ch(None, image=None, what_style="centered_text", window_style="centered_window", statement_name="say-centered", what_outlines=[ (2, "#260000", 0, 0), (1, "#d66", 0, 0) ], what_size=gui.text_size+8, what_font="font/pen.ttf", what_suffix="\n", what_line_spacing=4, what_kerning=1.5)


    hero = Ch(_("[hero_name!t]"), image="hero")
    hero_m = Ch(_("[hero_name!t]"), image="hero", what_italic=True, what_prefix="(- ", what_suffix=")")

    girl = Ch(_("Девушка"), color="#ccc", image="girl", callback=talk_callback)
    girl_m = Ch(_("Девушка"), color="#ccc", image="girl", what_italic=True, what_prefix="(- ", what_suffix=")") # 

    vv = Ch(_("Вампирша"), color="#c03", image="vv", callback=talk_callback)
    vv_m = Ch(_("Вампирша"), color="#c03", image="vv", what_italic=True, what_prefix="(- ", what_suffix=")")

    ss = Ch(_("Селена"), color="#c03", image="vv", callback=talk_callback)
    ss_m = Ch(_("Селена"), color="#c03", image="vv", what_italic=True, what_prefix="(- ", what_suffix=")")

    gg = Ch(_("[girl_name]"), color="#c03", image="vv", callback=talk_callback)
    gg_m = Ch(_("[girl_name]"), color="#c03", image="vv", what_italic=True, what_prefix="(- ", what_suffix=")")

    raz = Ch(_("Разбойник"), color="#9b3b3b")

    man1 = Ch(_("Мужчина 1"), color="#ccc")
    man2 = Ch(_("Мужчина 2"), color="#ccf")
    man3 = Ch(_("Мужчина 3"), color="#fcc")

    uu = Ch(_("Незнакомец"), color="#ccc")

    def normalize_who_name(name):
        if not name:
            return None

        aliases = {
            "hero_m": "hero",
            "girl_m": "girl",
            "vv_m": "vv",
            "ss_m": "ss",
            "gg_m": "gg",
        }

        return aliases.get(name, name)

    # автоматическое объявление изображений
    images_auto()

    # прятать окно при командах "pause", "hide", "show", "screen"
    # помимо команд по умолчанию: "scene", "call screen", "menu", "say-centered", "say-bubble"
    auto_hide()

    # вспышка
    def Flash(color="#fff", t=1):
        return Fade(t*.4, t*.2, t*.4, color=color)

    flash = Flash()

    # короткий fade
    ffade = Flash("#000", .25)

    # $ FF() - это with fade, но только если фон поменялся
    def FF(transition=ffade):
        if pnc_current_bg != pnc_last_bg:
            store.pnc_last_bg = pnc_current_bg
            renpy.transition(transition)
            renpy.restart_interaction()

    # дополняем стандартные функции отображения для работы с FF
    def pnc_scene(layer="master"):
        store.pnc_is_scene = True
        renpy.scene()

    def pnc_show(name, at_list=[ ], layer=None, what=None, zorder=0, tag=None, behind=[ ], atl=None, **kwargs):
        if pnc_is_scene and (str(name[0]) == "bg"):
            store.pnc_current_bg = str(" ".join(name))
            store.pnc_is_scene = False

        renpy.show(name, at_list, layer, what, zorder, tag, behind, atl, **kwargs)

    # жёсткая пауза
    def pause(t=.5):
        if t > 0: renpy.pause(t, hard=True)

define config.scene = pnc_scene
define config.show = pnc_show

default pnc_is_scene = False
default pnc_current_bg = None
default pnc_last_bg = None

# текстовое окно
define gui.textbox_height = 320
define gui.dialogue_ypos = 100
define gui.name_xpos = 277 + 44
define gui.name_text_size = 28

style window:
    background Transform("say textbox", align=(.5, 1.))

style namebox:
    background Frame("say namebox", 0, 50, 160, 50, ysize=100)
    left_padding 77
    right_padding 123
    ypadding 33

# кнопки quick_menu
style q_textbutton is textbutton:
    xysize (100, 100)
    pos (1550+50, 1010+50)
    anchor (.5, .5)
    background "button idle"

style q_textbutton_text is textbutton_text:
    layout "subtitle"
    align(.5, .5)
    text_align .5

# default persistent.quick_menu = False
define persistent.quick_menu = True

# кнопка для вызова quick_menu
screen quick_menu():
    zorder 100

    # on "show" action If(persistent.quick_menu and not has_screen("quick_line"), Show("quick_line"))

    # textbutton "📌" style "q_textbutton" action SetField(persistent, "quick_menu", not persistent.quick_menu) at transform:
        # subpixel True
        # alpha .75
        # matrixcolor BrightnessMatrix(0)

        # parallel:
            # on idle:
                # ease .2 zoom 1 matrixcolor BrightnessMatrix(0)

            # on hover, selected:
                # ease .2 zoom 1.25 matrixcolor BrightnessMatrix(.25)

    # add "quick_check"

    use quick_line

# полоса с кнопками из quick_menu
screen quick_line():
    zorder 99

    # imagebutton idle "pixel" action NullAction() alternate ShowMenu("gg_test")

    if not game_menu():
        frame:
            xalign .5
            background Frame(Transform("say quick", alpha=.5), 240, 24)
            padding (200, 16)
            xsize 1800
            yoffset 4

            at transform:
                subpixel True
                ypos 1.
                default yanchor 1.

                on show:
                    alpha 0 yanchor 0 
                    linear .2 alpha .75 yanchor 1.

                on hide:
                    linear .2 alpha 0 yanchor 0

            hbox:
                style_prefix "quick"
                style "quick_menu"

                # 📋📃⚙️🔎📌🛞📩📝✒️👁️👣❌⭕🔅🔆⚠️⏩⏪◀️✔️✖️🔙🔜
                # textbutton _("⏪ НАЗАД") action Rollback()
                # null width 32
                # textbutton _("ИСТОРИЯ 📃") action ShowMenu('history')
                # textbutton _("АВТО 🔜") action Preference("auto-forward", "toggle")
                # textbutton _("ВИД 👁️") action HideInterface()
                # textbutton _("СОХР️ ✒️") action ShowMenu('save')
                # textbutton _("ОПЦИИ ⚙️") action ShowMenu('preferences')
                # null width 32
                # textbutton _("ПРОПУСК ⏩") action Skip() alternate Skip(fast=True, confirm=True)

                textbutton _("НАЗАД") action Rollback()
                null width 32
                textbutton _("ИСТОРИЯ") action ShowMenu('history')

                if _skipping:
                    textbutton _("АВТО") action Preference("auto-forward", "toggle")
                    textbutton _("ВИД") action HideInterface()

                textbutton _("СОХР️") action ShowMenu('save')
                textbutton _("ОПЦИИ") action ShowMenu('preferences')

                if _skipping:
                    null width 32
                    textbutton _("ПРОПУСК") action Skip() alternate Skip(fast=True, confirm=True)

image pixel = Null(1, 1)

style main_menu_title:
    font "font/pen.ttf"

style mm_button is main_menu_button:
    xysize (535, 105)
    background "mm idle"
    hover_background "mm hover"
    selected_foreground "mm selected"

style mm_button_text is main_menu_button_text:
    size gui.text_size + 12
    hover_color "#fff"
    selected_color "#fff"
    align (.5, .5)
    text_align .5
    layout "subtitle"

default mm_link = "https://store.steampowered.com/app/3894300/"

init python:
    def mm_set_link_f(link, trans, st, at):
        store.mm_link = link
        renpy.restart_interaction()

# смена баннеров
define mm_link1 = "https://store.steampowered.com/app/3894300/"
define mm_link2 = "https://store.steampowered.com/app/4048970/"

# наведение на кнопку
transform mm_hover_at(t=.25, b=.1):
    matrixcolor BrightnessMatrix(0)

    on idle:
        ease t matrixcolor BrightnessMatrix(0)

    on hover:
        ease t matrixcolor BrightnessMatrix(b)

# видимость рекламного баннера при запуске игры = True
define mm_banner = True

screen main_menu():
    default mm_link_hover = False
    default mm_banner_page = 1
    default mm_dot_hover = 0
    tag menu
    add gui.main_menu_background
    # рамка с кнопками
    frame:
        background "mm overlay"
        xysize (670, 1080)

        vbox:
            spacing 8
            align (.5, .5)

            style_prefix "mm"

            textbutton _("Начать") action Start()
            textbutton _("Загрузить") action ShowMenu("load")
            textbutton _("Настройки") action ShowMenu("preferences")
            textbutton _("Об игре") action ShowMenu("about")

            if renpy.variant("pc") or (renpy.variant("web") and not renpy.variant("mobile")):
                textbutton _("Помощь") action ShowMenu("help")

            if renpy.variant("pc"):
                textbutton _("Выход") action Quit(confirm=not main_menu)

    # название и версия
    if gui.show_name:
        frame:
            style "empty"
            xysize (666, 205)

            text "[config.name!t]":
                style "main_menu_title"
                align (.5, .5)

        text "[config.version]":
            style "main_menu_version"
            anchor (1.0, 1.0)
            pos (1910, 1070)

    if mm_banner:

        fixed:
            xysize (491, 155)
            align (.061, .98)
            at mm_hover_at

            # Автосмена баннера каждые 6 секунд.
            timer 6.0 repeat True action [
                If(
                    mm_banner_page == 1,
                    SetScreenVariable("mm_banner_page", 2),
                    SetScreenVariable("mm_banner_page", 1)
                ),
                With(Dissolve(.35))
            ]

            button:
                background None
                padding (0, 0)
                xysize (491, 155)

                # Ссылка меняется вместе с текущим баннером.
                action OpenURL(mm_link1 if mm_banner_page == 1 else mm_link2)

                hovered SetScreenVariable("mm_link_hover", True)
                unhovered SetScreenVariable("mm_link_hover", False)

                add Solid("#000000" if mm_banner_page == 1 or mm_dot_hover == 1 else "#000000"):
                    xysize (497, 161)
                    align (.5, .5)

                viewport:
                    xysize (491, 155)
                    draggable False
                    mousewheel False

                    fixed:
                        xysize (491, 155)

                        if mm_banner_page == 1:
                            add "link1":
                                xysize (491, 155)

                        elif mm_banner_page == 2:
                            add "link2":
                                xysize (491, 155)

                        # Затемнение строго внутри рамки баннера.
                        if not mm_link_hover:
                            add Solid("#0008"):
                                xysize (491, 155)



            # Точки-переключатели внутри баннера.
            hbox:
                spacing 10
                align (.5, .97)

                # Первая точка
                button:
                    xysize (16, 16)
                    padding (0, 0)
                    margin (0, 0)

                    background Solid("#000000")
                    hover_background Solid("#000000")
                    insensitive_background Solid("#000000")

                    hovered SetScreenVariable("mm_dot_hover", 1)
                    unhovered SetScreenVariable("mm_dot_hover", 0)

                    action If(
                        mm_banner_page != 1,
                        [
                            SetScreenVariable("mm_banner_page", 1),
                            SetScreenVariable("mm_dot_hover", 0),
                            With(Dissolve(.35))
                        ],
                        NullAction()
                    )

                    add Solid("#ffffff" if mm_banner_page == 1 or mm_dot_hover == 1 else "#777777"):
                        xysize (12, 12)
                        align (.5, .5)

                # Вторая точка
                button:
                    xysize (16, 16)
                    padding (0, 0)
                    margin (0, 0)

                    background Solid("#000000")
                    hover_background Solid("#000000")
                    insensitive_background Solid("#000000")

                    hovered SetScreenVariable("mm_dot_hover", 2)
                    unhovered SetScreenVariable("mm_dot_hover", 0)

                    action If(
                        mm_banner_page != 2,
                        [
                            SetScreenVariable("mm_banner_page", 2),
                            SetScreenVariable("mm_dot_hover", 0),
                            With(Dissolve(.35))
                        ],
                        NullAction()
                    )

                    add Solid("#ffffff" if mm_banner_page == 2 or mm_dot_hover == 2 else "#777777"):
                        xysize (12, 12)
                        align (.5, .5)
            # imagebutton auto "btn x %s" xalign 1. action SetVariable("mm_banner", False), Function(renpy.restart_interaction)

    key "hide_windows" action SetVariable("mm_banner", not mm_banner), Function(renpy.restart_interaction)

# неубирающийся фон за слоем master
screen back(img="red"):
    layer "back"
    add img

# неубирающийся фон за слоем master
init python:
    renpy.add_layer("back", below="master", menu_clear=False)
    config.overlay_screens.append("back")

    # умножение с сохранением типа
    def imul(i, n):
        return i * n if isinstance(i, float) else int(i * n)

## ИНСТРУМЕНТЫ ДЛЯ ПРОЛИСТЫВАНИЯ ВВЕРХ
# скролл вверх с откатом
transform in_at(y0, y, dy=.05, t=show_t, w=0):
    subpixel True
    ypos y0
    (w if w and not renpy.is_skipping() else .0001)
    easeout (.0001 if renpy.is_skipping() else t * .55) ypos y + imul(y - y0, dy)
    easein (.0001 if renpy.is_skipping() else t * .2) ypos y - imul(y - y0, dy)
    easein (.0001 if renpy.is_skipping() else t * .15) ypos y + imul(y - y0, dy / 2)
    ease_back (.0001 if renpy.is_skipping() else t * .1) ypos y

# появление из-за правого края
transform in_right_at(dy=44, t=show_t, w=0):
    subpixel True
    xpos 1. xanchor 0 alpha 0 xoffset 0
    (w if w and not renpy.is_skipping() else .0001)
    easein (.0001 if renpy.is_skipping() else t * .5) xanchor .5 xpos .5 xoffset -dy alpha 1
    ease (.0001 if renpy.is_skipping() else t * .15) xoffset dy
    ease (.0001 if renpy.is_skipping() else t * .2) xoffset -dy * .5
    ease (.0001 if renpy.is_skipping() else t * .15) xoffset 0

# исчезание за левым краем
transform out_left_at(dy=.05, t=show_t, w=0):
    parallel:
        subpixel True
        xpos .5 xanchor .5
        (w if w and not renpy.is_skipping() else .0001)
        easeout (.0001 if renpy.is_skipping() else t * .15) xpos .5 + dy * .5
        ease (.0001 if renpy.is_skipping() else t * .2) xpos .5 - dy
        easein (.0001 if renpy.is_skipping() else t * .15) xpos .5 + dy
        easeout (.0001 if renpy.is_skipping() else t * .5) xanchor 1. xpos .0

    parallel:
        (w if w and not renpy.is_skipping() else .0001)
        (.0001 if renpy.is_skipping() else t * .5)
        linear (.0001 if renpy.is_skipping() else t * .5) alpha 0

# появление с наездом
transform in_slide_at(dx1=777, dy1=0, dx2=0, dy2=0, t=show_t):
    subpixel True
    offset (dx1, dy1) alpha 0
    easein (.0001 if renpy.is_skipping() else t) offset (dx2, dy2) alpha 1

# появление с наездом и тряской
transform in_elastic_at(dx1=777, dy1=0, dx2=0, dy2=0, t=show_t):
    parallel:
        subpixel True
        offset (dx1, dy1)
        ease_elastic (.0001 if renpy.is_skipping() else t) offset (dx2, dy2)

    parallel:
        subpixel True
        alpha 0
        linear (.0001 if renpy.is_skipping() else t * .5) alpha 1

# смещение с тряской
transform elastic_at(x=.5, y=0, x2=0, y2=0, t=show_t):
    subpixel True
    align (x, y)
    ease_elastic (.0001 if renpy.is_skipping() else t) align (x2, y2)

# сдвиг с рывком
transform aligning(x1=.5, y1=0, x2=0, y2=0, t=show_t):
    subpixel True
    align (x1, y1)
    ease (.0001 if renpy.is_skipping() else t) align (x2, y2)

# зум
transform zoom(z=1):
    subpixel True
    zoom z

# смена зума
transform zooming(z1=1.25, z2=1, t=show_t):
    subpixel True
    zoom z1
    linear (.0001 if renpy.is_skipping() else t) zoom z2

# смена зума с рывками
transform zooming2(z1=0, z2=1, t=show_t):
    subpixel True
    zoom z1
    ease_elastic (.0001 if renpy.is_skipping() else t) zoom z2

# положение
transform left2(x=.15):
    align (x, 0)

transform right2(x=.85):
    align (x, 0)

transform out_top_at(t=show_t):
    subpixel True
    ease (.0001 if renpy.is_skipping() else t) yanchor 1. ypos 0

transform out_bottom_at(t=show_t):
    subpixel True
    ease (.0001 if renpy.is_skipping() else t) yanchor 0 ypos 1.

init -2:
    transform align(x=.5, y=0):
        align (x, y)

    transform top0(x=.5, y=0, xa=.5):
        xanchor xa
        pos (x, y)

init -1 python:
    # показать картинку рывком снизу
    def show_in(tag, t=None):
        t = (show_t + .5) if t is None else t
        renpy.show(tag, at_list=[ in_at(config.screen_height, 0, t=t) ])
        renpy.restart_interaction()
        renpy.pause(t)

    # показать картинку наплывом справа
    def show_in_right(tag, t=None, scene=None, dx=777, dy=0, dx2=0, dy2=0, at=top0, elastic=False, **kwargs):
        t = (show_t + .5) if t is None else t

        if scene is None: scene = t or 1

        if scene:
            renpy.scene()

            if scene is True: renpy.with_statement(Dissolve(t))

        at_list = [ in_elastic_at(dx, dy, dx2, dy2, t) ] if elastic else [ in_slide_at(dx, dy, dx2, dy2, t) ]
        renpy.show(tag, what=Transform(tag, **kwargs), at_list=make_list(at) + at_list)

        if scene and not scene is True: renpy.transition(Dissolve(t))

        renpy.restart_interaction()
        renpy.pause(t)

    # сдвиг влево
    def sliding(tag, t=None, ax1=.5, ay1=0, ax2=0, ay2=0, at=None, **kwargs):
        t0 = t
        t = (show_t + .5) if t is None else t
        renpy.show(tag, what=Transform(tag, **kwargs), at_list=make_list(at) + [ elastic_at(ax1, ay1, ax2, ay2, t or show_t + .5) ])
        renpy.restart_interaction()
        t = .0001 if t0 == 0 else t
        renpy.pause(t)

    # сдвиг влево с рывками
    def elasting(tag, t=None, ax1=.5, ay1=0, ax2=0, ay2=0, at=[ ], **kwargs):
        t0 = t
        t = (show_t + .5) if t is None else t
        renpy.show(tag, what=Transform(tag, **kwargs), at_list=make_list(at) + [ elastic_at(ax1, ay1, ax2, ay2, t or show_t + .5) ])
        renpy.restart_interaction()
        renpy.pause(.0001 if t0 == 0 else t)

    # показать картинку наплывом слева
    def show_in_left(tag, t=None, scene=True, dx=-999, dy=0, dx2=0, dy2=0, at=top0, elastic=False, **kwargs):
        show_in_right(tag, t, scene, dx, dy, dx2, dy2, at, elastic, **kwargs)

    # показать картинку наплывом сверху
    def show_in_top(tag, t=None, scene=True, dx=0, dy=-999, dx2=0, dy2=0, at=top0, elastic=False, **kwargs):
        show_in_right(tag, t, scene, dx, dy, dx2, dy2, at, elastic, **kwargs)

    # показать картинку наплывом снизу
    def show_in_bottom(tag, t=None, scene=True, dx=0, dy=999, dx2=0, dy2=0, at=align(.5, 1.), elastic=False, **kwargs):
        show_in_right(tag, t, scene, dx, dy, dx2, dy2, at, elastic, **kwargs)

    # вытеснение появлением из-за правого края
    def in_right(tag, t=None):
        t = (show_t + .5) if t is None else t

        all = [ str(i) for i in renpy.get_showing_tags() if str(i).startswith("ch_") ]

        if all:
            for i in all: renpy.show(i, at_list=[ out_left_at(t=(t or show_t)) ])

            renpy.restart_interaction()

            if t: renpy.pause(t)

        renpy.show(tag, at_list=[ in_right_at(t=(t or show_t)) ])

        if t: renpy.pause(t)

    # показать картинку, сдвинув остальные вверх
    def push_in(tag, dy=config.screen_height, t=None, at=None, **kwargs):
        if t is None: t = show_t

        # размеры новой картинки
        w, h = renpy.render(renpy.displayable(tag), config.screen_width, config.screen_height, 0, 0).get_size()

        # высота новой картинки с отступом
        h = int(h + pad_bottom) if dy is None else int(dy)

        # все комиксные картинки на экране (из глав - начинаются на ch_)
        all = [ str(i) for i in renpy.get_showing_tags() if str(i).startswith("ch_") ]

        ymax = 0

        for i in all:
            x0, y0, w0, h0 = renpy.get_image_bounds(i)
            ymax = max(ymax, int(y0 + h0))

        hh = h

        if dy is None:
            hh = int(ymax - config.screen_height + h)

            if hh < 0: h, hh = config.screen_height - ymax, 0

        if hh:
            for i in all:
                x0, y0, w0, h0 = renpy.get_image_bounds(i)
                renpy.show(i, at_list=[ in_at(int(y0), int(y0 - hh)) ])

        # выдвинуть новую картинку
        renpy.show(tag, what=Transform(tag, **kwargs), at_list=make_list(at)+[ in_at(config.screen_height, int(config.screen_height - h)) ])
        renpy.restart_interaction()

        if t: renpy.pause(t)

# эффект для смены "говорящих голов" в текстбоксе с помощью xoffset
transform side_image_change_transform(old, new):
    contains:
        subpixel True
        xanchor .5
        xpos 320
        yalign 1.
        yzoom 1
        old
        xzoom 1
        linear .25 xzoom 0

    contains:
        subpixel True
        xanchor .5
        xpos 320
        yalign 1.
        yzoom 1
        new
        xzoom 0
        linear .25 xzoom 1

# если меняются только эмоции у того же персонажа
transform side_image_same_transform(old, new):
    contains:
        subpixel True
        xanchor .5
        xpos 320
        yalign 1.
        yzoom 1
        old
        new with Dissolve(.25, alpha=True, mipmap=True)

init -2:
    transform show_hide(t=.2):
        on show:
            alpha 0
            linear t alpha 1

        on hide:
            linear t alpha 0

    transform only_show(t=.2):
        on show:
            alpha 0
            linear t alpha 1

    transform only_hide(t=.2):
        on hide:
            linear t alpha 0

    # реакция на курсор
    transform hover_at(t=.2, z=1.25, b=.25):
        subpixel True
        anchor (.5, .5)

        on idle:
            ease t zoom 1 matrixcolor BrightnessMatrix(0)

        on hover, selected:
            ease t zoom z matrixcolor BrightnessMatrix(b)

# применяем эффект для смены "говорящих голов" в текстбоксе
define config.side_image_change_transform = side_image_change_transform

# эффект для случаев, когда сам персонаж не меняется, только его эмоции
define config.side_image_same_transform = side_image_same_transform

# для грома
# default thunder_sound_i = 1
# default thunder_next_st = rndf(3, 5)

# невидимая картинка с рандомными раскатами грома
# image thunder = DynamicDisplayable(thunder_f)

init -2 python:
    # включить звук дождя и грома
    # def rain_on():
        # sfxplay("rain")
        # renpy.show("thunder", layer="back")
        # renpy.restart_interaction()

    # выключить звук дождя и грома
    # def rain_off():
        # sfxstop()
        # renpy.hide("thunder", layer="back")
        # renpy.restart_interaction()

    # время нарастания-затухания песен
    default_fade = 2

    # добавить при необходимости путь
    def add_dir(fn, dir):
        if fn.startswith("<") or not (dir and fn): return fn

        if not dir.endswith("/"): dir = dir + "/"

        return fn if fn.startswith(dir) else dir + fn

    # добавить при необходимости расширение
    def add_ext(fn, ext):
        if fn.endswith(">") or not (ext and fn): return fn

        if not ext.startswith("."): ext = "." + ext

        return fn if fn.endswith(ext) else fn + ext

    # randint для float
    def rndf(x1, x2):
        x1, x2 = min(x1, x2), max(x1, x2)

        return x1 + renpy.random.random() * (x2 - x1)

    def rnds(*args):
        return renpy.random.choice(args)

    # канал для зацикленного эффекта
    renpy.music.register_channel("ambient", "ambient", loop=True, tight=True)

    # воспроизвести звук или список звуков (не config.play_channel)
    def splay(mname, fadein=0, fadeout=0, channel="audio", ext="ogg"):
        if mname:
            renpy.sound.play([ add_ext(add_dir(i, "audio"), ext) for i in make_list(mname) ], channel=channel, fadein=fadein, fadeout=fadeout)

    # воспроизвести звук или список звуков на канале sound
    def loopplay(mname, fadein=0, fadeout=0, channel="sound", ext="ogg"):
        if mname:
            renpy.sound.play([ add_ext(add_dir(i, "audio"), ext) for i in make_list(mname) ], channel=channel, fadein=fadein, fadeout=fadeout, loop=True)

    # остановка звука
    def loopstop(fadeout=default_fade, channel="sound"):
        renpy.music.stop(channel=channel, fadeout=fadeout)

    # фоновый зацикленный эффект
    def sfxplay(mname, fadein=.5, fadeout=.5, channel="ambient", ext="ogg", folder="ambient"):
        # renpy.music.set_volume(volume, 0, channel)
        if mname: renpy.music.play([ add_ext(add_dir(i, folder), ext) for i in make_list(mname) ], channel=channel, fadein=fadein, fadeout=fadeout, loop=True)

    # остановка фонового эффекта
    def sfxstop(fadeout=.5, channel="ambient"):
        renpy.sound.stop(channel=channel, fadeout=fadeout)

    # запустить музыку или плейлист
    def mplay(mname, fadein=default_fade, fadeout=default_fade, loop=True, channel="music", ext="ogg", music_dir="music"):
        if mname: renpy.music.play([ add_ext(add_dir(i, music_dir), ext) for i in make_list(mname) ], channel=channel, loop=loop, fadein=fadein, fadeout=fadeout)

    # остановка музыки
    def mstop(fadeout=default_fade, channel="music"):
        renpy.music.stop(channel=channel, fadeout=fadeout)

    # гром
    # def thunder_f(st, at):
        # global thunder_sound_i, thunder_next_st

        # if st >= thunder_next_st:
            # thunder_next_st = st + rndf(3, 7)
            # i = thunder_sound_i

            # while i == thunder_sound_i: i = rnds(1, 2, 3)

            # thunder_sound_i = i
            # splay("thunder%s" % (thunder_sound_i))

        # return "null", 1. / config.framerate

    # дополняем список команд, при запуске которых текстбокс будет прятаться в режиме config.window = "auto"
    # добвляется к списку по умолчанию: "scene", "call screen", "menu", "say-centered", "say-bubble"
    def auto_hide(*args):
        if not args: args = [ "pause", "hide", "show", "screen" ]

        for i in make_list(args):
            if not i in config.window_auto_hide: config.window_auto_hide.append(i)

# БЛОК ДЛЯ РАБОТЫ С ВРЕМЕНАМИ СУТОК
# как пользоваться:

# 1) если есть нужные рисунки, то назвать файлы по временам суток:
# bg_street_night, bg_street_morning и т.д. по списку
# в противном случае будет перекрашиваться основная картинка
# например, "bg street day" или скопированная в неё "bg street"
# первый суффикс в списке времён суток можно и не указывать: bg street

# 2) воспользоваться автоматическим объявлением спрайтов в блоке init:
# $ images_auto()

# 3) указать перфиксы спрайтов, которые будут зависеть от времени суток
# например, чтобы все фоны и все спрайты eileen и pytom зависели от времени суток:
# daytime_prefix = [ "bg", "eileen", "pytom" ]

# 4) при необходимости задать список времён суток, по умолчанию он:
# alldaytime = [ "day", "night" ]

# 5) в скрипте просто задавать время суток $ setdaytime("night")
# или переключение на следующее по списку (зациклено): $ setdaytime()
# спрайты на экране сразу поменяются

init -222:
    # настройки нового освещения, на входе сразу матрица
    transform daytime_light(matrix):
        matrixcolor matrix

    transform daytime_empty():
        pass

    # привязать положение какого-нибудь объекта к курсору
    # якорь всегда в центре
    transform xy_at:
        function xy_at_f

init -99 python:
    # если это не список, то сделать единственным элементом списка
    def make_list(param, none=[ ]):
        if param is None: return none

        if not isinstance(param, list): param = [ param ]

        return param

    # автоматическое объявление изображений (вызывается из init)
    def images_auto(folders=[ ], spaces=[ ' ', '_', '/' ], minimum=1, dlc="DLC"):
        # если задано имя папки, то нумеруем сотню копий и добавляем к проверяемым папкам
        # если dlc == "DLC", то ищем "DLC1", "DLC2", "DLC3" и т.д.
        folders = [ "images" ] + make_list(folders)
        if isinstance(dlc, (str, unicode)):
            folders = folders + [ str(dlc) + str(i) for i in range(1, 101) ]
        for i in folders:
            if not i in config.automatic_images_strip:
                config.automatic_images_strip.append(i)
        config.automatic_images = spaces
        config.automatic_images_minimum_components = minimum

    # узнать имя и цвет персонажа
    def get_name_color(char, color=None):
        if color is None:
            color = gui.accent_color

        col = color

        if isinstance(char, (str, unicode)):
            name = char
        else:
            name = char.name

            for arg in [ "color", "who_color" ]:
                if arg in char.who_args.keys():
                    col = char.who_args[arg]

        return name, col

    # список допустимых времён суток
    alldaytime = [ "day", "night" ]

    # список префиксов-меток для динамических спрайтов,
    # освещение которых будет зависеть от времени суток в curdaytime
    # например, все фоны и пара героев:
    # daytime_prefix = ["bg", "eileen", "pytom"]
    # к таким именам будет добавлен суффикс из daytime_suffix
    # а потом они будут превращены в динамические уже со своими именами без суффикса
    daytime_prefix = [ "bg" ]

    # по умолчанию первое в списке время суток
    curdaytime = alldaytime[0]

    # суффикс для динамических спрайтов - первое в списке время суток
    daytime_suffix = alldaytime[0]

    # здесь будут храниться имена изменённых спрайтов, зависимых от curdaytime
    daytime_suffixed = [ ]

    # префикс-метка для фонов
    bg_prefix = "bg"

    # переключение времени суток на новое или на следующее, если не указывать, на какое
    def setdaytime(newdaytime=None, effect=dissolve):
        if newdaytime is None:
            # если новое время суток == None,
            # то переключаемся на следующее в списке (по кругу)
            i = alldaytime.index(curdaytime) + 1

            if i >= len(alldaytime): i = 0

            newdaytime = alldaytime[i]

        if effect:
            renpy.show("black", tag="daytimeblack")
            renpy.with_statement(effect)

        store.curdaytime = newdaytime

        if effect:
            renpy.hide("daytimeblack")
            renpy.with_statement(effect)

    # получить трансформ с матрицей для ночного освещения
    # по умолчанию для спрайта, True - для фона
    def atdaytime(bg=False):
        if bg:
            key = "_" + bg_prefix + "_"
        else:
            key = "_"
        key = curdaytime + key + "attrs"
        attrs = ("#000", 0, 1, 0)
        if key in globals().keys():
            color, brightness, saturation, contrast = globals()[key]
            matrix = BrightnessMatrix(brightness) * ContrastMatrix(contrast) * TintMatrix(color) * SaturationMatrix(saturation)
            return daytime_light(matrix)
        return daytime_empty()

    # все поддерживаемые расширения картинок
    all_image_extensions = [ ".png", ".jpg", ".jpeg", ".webp" ]

    # все поддерживаемые расширения видео
    all_video_extensions = [ ".webm", ".ogv", ".mp4" ]

    # можно задать дополнительные параметры для конкретных картинок
    # например, в блоке init python можно по умолчанию сдвинуть eileen happy вниз:
    # images_auto_kwargs["eileen happy"] = { "yoffset": 20 }
    images_auto_kwargs = { }

    # получить расширение из тех, что есть на входе, на которое оканчивается строка s
    # case=False - нечувствительно к регистру
    def endswith(s, exts=None, case=False):
        if exts is None: exts = all_image_extensions + all_video_extensions

        for i in exts:
            t = s if case else s.lower()

            if t.endswith(i): return i

        return None

    # смешивание нескольких словарей
    def mix_dicts(*args):
        x = args[0].copy()

        for i in range(1, len(args)):
            x.update(args[i])

        return x

# БЛОК ДЛЯ ДОПОЛНЕННОГО АВТООБЪЯВЛЕНИЯ СПРАЙТОВ
# стандартное автообъявление картинок, но с webp
# плюс не разбиваются на теги имена, предназначенные для LayerdImage
# вместо этого объединяются через нижний минус "_"
# префиксы, по которым определяются такие имена берутся из layered_prefixes
# выполняется после всего остального
init 1900 python hide:
    def create_automatic_images():
        seps = config.automatic_images

        if seps is True:
            seps = [ ' ', '/', '_' ]

        for dir, fn in renpy.loader.listdirfiles():

            if fn.startswith("_"): continue

            # только .png и .jpg и .jpeg и .webp
            if not endswith(fn): continue

            # убираем расширения и заменяем слеши.
            ext = endswith(fn)
            shortfn = fn[:-len(ext)].replace("\\", "/")

            # делим строку на части
            name = ( shortfn, )

            for sep in seps: name = tuple(j for i in name for j in i.split(sep))

            # выбрасываем имя папок из тегов
            while name:
                for i in config.automatic_images_strip:
                    if name[0] == i:
                        name = name[1:]
                        break

                else:
                    break

            # для проверки префиксов и суффиксов после
            # возможной склейки тегов
            prefix = name[0]
            suffix = name[len(name) - 1]

            # убираем суффикс, если это первое время суток
            if suffix == daytime_suffix: name = name[:-1]

            # имя без суффикса
            name0 = name

            # теги одной строкой
            sname = " ".join(name)

            # не делим на части имена, которые начинаются с префиксов для LayeredImage
            layered = False

            if prefix in layered_prefixes:
                name = "_".join(name)
                sname = name
                layered = True

            # добавляем суффикс по умолчанию, если начинается
            # с префикса из списка меток для динамических спрайтов,
            # которые будут менять освещение в зависимости от времени суток
            if prefix in daytime_prefix:
                # нельзя автоматизировать файлы с суффиксами времён суток,
                # потому что они станут частью автоматизированного спрайта
                if not suffix in alldaytime[1:]:
                    if not sname in daytime_suffixed:
                        # запоминаем имя изменённого спрайта
                        store.daytime_suffixed.append(sname)

                        # добавляем суффикс
                        if layered:
                            name = name + " " + daytime_suffix

                        else:
                            name = name + (daytime_suffix,)

            # игнорируем, если уже создана одноимённая копия с суффиксом
            if name0 in daytime_suffixed: continue

            # игнорируем, если не набирается указанное в переменной количество тегов
            if len(name) < config.automatic_images_minimum_components: continue

            # игнорируем, если такой спрайт уже есть
            if name in renpy.display.image.images: continue

            # если тег один, но требуется для многослойных картинок, то добавляем в конце "_"
            if layered and not "_" in name: name = name + "_"

            # дополнительные параметры
            kwargs = images_auto_kwargs.get(None, { })

            if name in images_auto_kwargs.keys(): kwargs = mix_dicts(kwargs, images_auto_kwargs[name])

            # объявляем видео-спрайт
            if ext in all_video_extensions:
                renpy.image(name, Movie(play=fn, **kwargs))

            # объявляем обычный спрайт
            else:
                renpy.image(name, fn, **kwargs)

    # если заданы параметры, то объявляем спрайты автоматом
    if config.automatic_images:
        create_automatic_images()

    # функция для динамического освещения спрайтов и фонов
    def def_daytime(st, at, img):
        # ищем картинку для текущего времени суток
        new = img + " " + curdaytime

        if has_image(new): return new, None

        # если не нашли, то берем основную
        new = img + " " + daytime_suffix

        return new, None

    # если заданы параметры, то создаём динамически освещённые спрайты
    if len(daytime_suffixed) > 0:
        # перебираем все спрайты, для которых нужна реакция на время суток
        for i in daytime_suffixed: renpy.image(i, DynamicDisplayable(def_daytime, i))
