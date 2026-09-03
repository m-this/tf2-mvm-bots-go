/*
Package loadouts is source/redbots3/loadouts.sp: what a defender bot carries.

The pools are the plugin's own, index for index. The command that remembers an
upgrade and the timer that hands the loadout out stay in the plugin, because a
console command and a timer are function references and the subset has none.
*/
package loadouts

// MaxRuntimeAttributes is how many attributes one weapon may carry back from
// the upgrade station.
//
//sp:name MAX_RUNTIME_ATTRIBUTES
const MaxRuntimeAttributes = 20

/*
ItemDefDefault is the item the class starts with, which has no definition index
of its own.

Not emitted as TF_ITEMDEF_DEFAULT: the plugin declares that one behind a guard,
because tf_econ_data declares it too when it is included. Same number, and this
one is ours.
*/
const ItemDefDefault = -1

//sp:name WEAPONS_SCOUT_PRIMARY
var weaponsScoutPrimary = [31]int32{
	ItemDefDefault, 200, 45, 220, 448, 669, 772, 799, 808, 888, 897, 906, 915, 964, 973, 1078,
	1103, 15002, 15015, 15021, 15029, 15036, 15053, 15065, 15069, 15106, 15107, 15108, 15131,
	15151, 15157,
}

//sp:name WEAPONS_SCOUT_SECONDARY
var weaponsScoutSecondary = [27]int32{
	ItemDefDefault, 209, 46, 160, 163, 222, 294, 449, 773, 812, 833, 1121, 1145, 15013, 15018,
	15035, 15041, 15046, 15056, 15060, 15061, 15100, 15101, 15102, 15126, 15148, 30666,
}

//sp:name WEAPONS_SCOUT_MELEE
var weaponsScoutMelee = [27]int32{
	ItemDefDefault, 190, 44, 221, 264, 317, 325, 349, 355, 423, 450, 452, 474, 572, 648, 660, 880,
	939, 954, 999, 1013, 1071, 1123, 1127, 1202, 30667, 30758,
}

//sp:name WEAPONS_SOLDIER_PRIMARY
var weaponsSoldierPrimary = [31]int32{
	ItemDefDefault, 205, 127, 228, 414, 441, 513, 658, 730, 800, 809, 889, 898, 907, 916, 965,
	974, 1085, 1104, 15006, 15014, 15028, 15043, 15052, 15057, 15081, 15104, 15105, 15129, 15130,
	15150,
}

//sp:name WEAPONS_SOLDIER_SECONDARY
var weaponsSoldierSecondary = [22]int32{
	ItemDefDefault, 199, 129, 133, 226, 354, 415, 442, 444, 1001, 1101, 1141, 1153, 15003, 15016,
	15044, 15047, 15085, 15109, 15132, 15133, 15152,
}

//sp:name WEAPONS_SOLDIER_MELEE
var weaponsSoldierMelee = [19]int32{
	ItemDefDefault, 196, 128, 154, 264, 357, 416, 423, 447, 474, 775, 880, 939, 954, 1013, 1071,
	1123, 1127, 30758,
}

//sp:name WEAPONS_PYRO_PRIMARY
var weaponsPyroPrimary = [31]int32{
	ItemDefDefault, 208, 40, 215, 594, 659, 741, 798, 807, 887, 896, 905, 914, 963, 972, 1146,
	1178, 15005, 15017, 15030, 15034, 15049, 15054, 15066, 15067, 15068, 15089, 15090, 15115,
	15141, 30474,
}

//sp:name WEAPONS_PYRO_SECONDARY
var weaponsPyroSecondary = [21]int32{
	ItemDefDefault, 199, 39, 351, 415, 595, 740, 1081, 1141, 1153, 1179, 1180, 15003, 15016,
	15044, 15047, 15085, 15109, 15132, 15133, 15152,
}

//sp:name WEAPONS_PYRO_MELEE
var weaponsPyroMelee = [26]int32{
	ItemDefDefault, 192, 38, 153, 214, 264, 326, 348, 423, 457, 466, 474, 593, 739, 813, 834, 880,
	939, 954, 1000, 1013, 1071, 1123, 1127, 1181, 30758,
}

//sp:name WEAPONS_DEMOMAN_PRIMARY
var weaponsDemomanPrimary = [17]int32{
	ItemDefDefault, 206, 308, 405, 608, 996, 1007, 1101, 1151, 15077, 15079, 15091, 15092, 15116,
	15117, 15142, 15158,
}

//sp:name WEAPONS_DEMOMAN_SECONDARY
var weaponsDemomanSecondary = [31]int32{
	ItemDefDefault, 207, 130, 131, 265, 406, 661, 797, 806, 886, 895, 904, 913, 962, 971, 1099,
	1144, 1150, 15009, 15012, 15024, 15038, 15045, 15048, 15082, 15083, 15084, 15113, 15137,
	15138, 15155,
}

//sp:name WEAPONS_DEMOMAN_MELEE
var weaponsDemomanMelee = [24]int32{
	ItemDefDefault, 191, 132, 154, 172, 264, 266, 307, 327, 357, 404, 423, 474, 482, 609, 880,
	939, 954, 1013, 1071, 1082, 1123, 1127, 30758,
}

//sp:name WEAPONS_HEAVY_PRIMARY
var weaponsHeavyPrimary = [34]int32{
	ItemDefDefault, 202, 41, 298, 312, 424, 654, 793, 802, 811, 832, 850, 882, 891, 900, 909, 958,
	967, 1206, 15004, 15020, 15026, 15031, 15040, 15055, 15086, 15087, 15088, 15098, 15099, 15123,
	15124, 15125, 15147,
}

//sp:name WEAPONS_HEAVY_SECONDARY
var weaponsHeavySecondary = [14]int32{
	ItemDefDefault, 199, 425, 1141, 1153, 15003, 15016, 15044, 15047, 15085, 15109, 15132, 15133,
	15152,
}

//sp:name WEAPONS_HEAVY_MELEE
var weaponsHeavyMelee = [22]int32{
	ItemDefDefault, 195, 43, 239, 264, 310, 331, 423, 426, 474, 587, 656, 880, 939, 1013, 1071,
	1084, 1100, 1123, 1127, 1184, 30758,
}

//sp:name WEAPONS_ENGINEER_PRIMARY
var weaponsEngineerPrimary = [18]int32{
	ItemDefDefault, 199, 141, 527, 588, 997, 1004, 1141, 1153, 15003, 15016, 15044, 15047, 15085,
	15109, 15132, 15133, 15152,
}

//sp:name WEAPONS_ENGINEER_SECONDARY
var weaponsEngineerSecondary = [23]int32{
	ItemDefDefault, 209, 140, 160, 294, 528, 1086, 1202, 15013, 15018, 15035, 15041, 15046, 15056,
	15060, 15061, 15100, 15101, 15102, 15126, 15148, 30666, 30668,
}

//sp:name WEAPONS_ENGINEER_MELEE
var weaponsEngineerMelee = [27]int32{
	ItemDefDefault, 197, 142, 155, 169, 329, 423, 589, 662, 795, 804, 884, 893, 902, 911, 960,
	969, 1071, 1123, 15073, 15074, 15075, 15139, 15140, 15114, 15156, 30758,
}

//sp:name WEAPONS_MEDIC_PRIMARY
var weaponsMedicPrimary = [6]int32{
	ItemDefDefault, 204, 36, 305, 412, 1079,
}

//sp:name WEAPONS_MEDIC_SECONDARY
var weaponsMedicSecondary = [26]int32{
	ItemDefDefault, 211, 35, 411, 663, 796, 805, 885, 894, 903, 912, 961, 970, 998, 15008, 15010,
	15025, 15039, 15050, 15078, 15097, 15121, 15122, 15120, 15145, 15146,
}

//sp:name WEAPONS_MEDIC_MELEE
var weaponsMedicMelee = [19]int32{
	ItemDefDefault, 198, 37, 173, 264, 304, 413, 423, 474, 880, 939, 954, 1003, 1013, 1071, 1123,
	1127, 1143, 30758,
}

//sp:name WEAPONS_SNIPER_PRIMARY
var weaponsSniperPrimary = [35]int32{
	ItemDefDefault, 201, 56, 230, 402, 526, 664, 752, 792, 801, 851, 881, 890, 899, 908, 957, 966,
	1005, 1092, 1098, 15000, 15007, 15019, 15023, 15033, 15059, 15070, 15071, 15072, 15111, 15112,
	15135, 15136, 15154, 30665,
}

//sp:name WEAPONS_SNIPER_SECONDARY
var weaponsSniperSecondary = [19]int32{
	ItemDefDefault, 203, 57, 58, 231, 642, 751, 1083, 1105, 1149, 15001, 15022, 15032, 15037,
	15058, 15076, 15110, 15134, 15153,
}

//sp:name WEAPONS_SNIPER_MELEE
var weaponsSniperMelee = [16]int32{
	ItemDefDefault, 193, 171, 232, 264, 401, 423, 474, 880, 939, 954, 1013, 1071, 1123, 1127,
	30758,
}

//sp:name WEAPONS_SPY_SECONDARY
var weaponsSpySecondary = [20]int32{
	ItemDefDefault, 210, 61, 161, 224, 460, 525, 1006, 1142, 15011, 15027, 15042, 15051, 15062,
	15063, 15064, 15103, 15128, 15127, 15149,
}

//sp:name WEAPONS_SPY_BUILDING
var weaponsSpyBuilding = [7]int32{
	ItemDefDefault, 736, 810, 831, 933, 1080, 1102,
}

//sp:name WEAPONS_SPY_MELEE
var weaponsSpyMelee = [29]int32{
	ItemDefDefault, 194, 225, 356, 423, 461, 574, 638, 649, 665, 727, 794, 803, 883, 892, 901,
	910, 959, 968, 1071, 15080, 15094, 15095, 15096, 15118, 15119, 15143, 15144, 30758,
}

//sp:name WEAPONS_SPY_PDA2
var weaponsSpyPda2 = [7]int32{
	ItemDefDefault, 212, 59, 60, 297, 947, 1205,
}
