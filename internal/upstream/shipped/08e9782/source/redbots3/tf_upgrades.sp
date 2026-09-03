#if defined METHOD_MVM_UPGRADES

/* A ceiling on the number of upgrades, not the number of them

The count comes from the game, through UpgradeCount below. This is only the point past which the
manager is not what we think it is and walking further would read memory that is not a list of
upgrades. */
#define MAX_UPGRADES	128

/* What the game held when this was last measured, for when it will not say

sm_dump_upgrades walked CMannVsMachineUpgradeManager and printed sixty three. The constant here
used to be sixty two and was used as the loop bound, so the last upgrade the game holds was one
no loop in this mod ever reached. */
#define UPGRADE_COUNT_MEASURED	63

//Size of attribute string
#define MAX_ATTRIBUTE_DESCRIPTION_LENGTH	128

enum //CEconItemAttributeDefinition
{
	m_pKVAttribute = 0,
	m_nDefIndex = 4
}

methodmap CEconItemAttributeDefinition
{
	property Address Address
	{
		public get() { return view_as<Address>(this); }
	}

	public int GetIndex()
	{
		int iAttribIndex = LoadFromAddress(this.Address + view_as<Address>(m_nDefIndex), NumberType_Int32);
		
		if (iAttribIndex > 3018 || iAttribIndex < 0)
			iAttribIndex = LoadFromAddress(this.Address - view_as<Address>(m_nDefIndex), NumberType_Int32); 
	
		return iAttribIndex;
	}
}

public CEconItemAttributeDefinition CEIAD_GetAttributeDefinitionByName(const char[] szAttribute)
{
	Address CEconItemSchema = GEconItemSchema();
	
	if (CEconItemSchema == Address_Null)
		return view_as<CEconItemAttributeDefinition>(Address_Null);
		
	return view_as<CEconItemAttributeDefinition>(GetAttributeDefinitionByName(CEconItemSchema, szAttribute));
}

//CMannVsMachineUpgrades
static int offset_flCap;
static int offset_nUIGroup;
static int offset_nTier;
static int CMannVsMachineUpgrades_Size;

enum //CMannVsMachineUpgradeManager
{
	m_Upgrades = 12, //0x000C
	
	CMannVsMachineUpgradeManager_Size = 28
} //Size=0x001C

methodmap CMannVsMachineUpgrades
{
	property Address Address
	{
		public get() 
		{
			return view_as<Address>(this);
		}
	}
	
	public char[] m_szAttribute()
	{
		//szAttrib is located at 0, no need to add its offset here
		
		char attribute[MAX_ATTRIBUTE_DESCRIPTION_LENGTH];
		
		for (int i = 0; i < sizeof(attribute); i++)
			attribute[i] = LoadFromAddress(this.Address + view_as<Address>(i), NumberType_Int32);
		
		return attribute;
	}
	
	public float m_flCap()
	{
		return float(LoadFromAddress(this.Address + view_as<Address>(offset_flCap), NumberType_Int32));
	}
	
	public int m_iUIGroup()
	{
		return LoadFromAddress(this.Address + view_as<Address>(offset_nUIGroup), NumberType_Int32);
	}
}

methodmap CMannVsMachineUpgradeManager < CMannVsMachineUpgrades
{
	public CMannVsMachineUpgradeManager() 
	{
		return view_as<CMannVsMachineUpgradeManager>(g_pMannVsMachineUpgrades);
	}
	
	/* How many upgrades the game actually holds

	m_Upgrades is a CUtlVector, and GetUpgradeByIndex below reads its m_pMemory at offset zero to
	find the elements. m_Size is the fourth int of that structure, after the pointer, the
	allocation count and the growth size, which puts it twelve bytes in.

	Read rather than assumed, because the whole point of asking is that nobody should be counting
	lines in a text file and hoping the game agrees. */
	public int Count()
	{
		return LoadFromAddress(this.Address + view_as<Address>(m_Upgrades + 12), NumberType_Int32);
	}
	
	public CMannVsMachineUpgrades GetUpgradeByIndex(int index)
	{
		Address rawUpgrades = this.Address + view_as<Address>(m_Upgrades);
		Address pUpgrades = DereferencePointer(rawUpgrades);
		
		return view_as<CMannVsMachineUpgrades>(pUpgrades + view_as<Address>(index * CMannVsMachineUpgrades_Size));
	}
}

//How many upgrades the game holds, asked of the game rather than counted in a text file
int UpgradeCount()
{
	int count = CMannVsMachineUpgradeManager().Count();
	
	if (count < 1 || count > MAX_UPGRADES)
		return UPGRADE_COUNT_MEASURED;
	
	return count;
}

void InitMvMUpgrades(GameData hGamedata)
{
	offset_flCap = hGamedata.GetOffset("CMannVsMachineUpgrades::flCap");
	offset_nUIGroup = hGamedata.GetOffset("CMannVsMachineUpgrades::nUIGroup");
	offset_nTier = hGamedata.GetOffset("CMannVsMachineUpgrades::nTier");
	CMannVsMachineUpgrades_Size = offset_nTier + 4;
	
#if defined TESTING_ONLY
	LogMessage("InitMvMUpgrades: CMannVsMachineUpgrades->flCap = %d", offset_flCap);
	LogMessage("InitMvMUpgrades: CMannVsMachineUpgrades->nUIGroup = %d", offset_nUIGroup);
	LogMessage("InitMvMUpgrades: CMannVsMachineUpgrades->nTier = %d", offset_nTier);
	LogMessage("InitMvMUpgrades: Size of CMannVsMachineUpgrades = %d", CMannVsMachineUpgrades_Size);
#endif
}

/* TECHNICAL DATA FOR REFERENCE
class CEconItemAttributeDefinition
{
	KeyValues	*m_pKVAttribute;
	attrib_definition_index_t	m_nDefIndex;
	const class ISchemaAttributeType *m_pAttrType;
	bool		m_bHidden;
	bool		m_bWebSchemaOutputForced;
	bool		m_bStoredAsInteger;
	bool		m_bInstanceData;
	EAssetClassAttrExportRule_t	m_eAssetClassAttrExportRule;
	uint32		m_unAssetClassBucket;
	bool		m_bIsSetBonus;
	int			m_iUserGenerationType;
	attrib_effect_types_t m_iEffectType;
	int			m_iDescriptionFormat;
	const char	*m_pszDescriptionString;
	const char	*m_pszArmoryDesc;
	const char	*m_pszDefinitionName;
	const char	*m_pszAttributeClass;
	bool		m_bCanAffectMarketName;
	bool		m_bCanAffectRecipeComponentName;
	econ_tag_handle_t	m_ItemDefinitionTag;
	mutable string_t	m_iszAttributeClass;
}

class CMannVsMachineUpgrades
{
	char szAttrib[ MAX_ATTRIBUTE_DESCRIPTION_LENGTH ];
	char szIcon[ MAX_PATH ];
	float flIncrement;
	float flCap;
	int nCost;
	int nUIGroup;
	int nQuality;
	int nTier;
}

class CMannVsMachineUpgradeManager
{
	CUtlVector< CMannVsMachineUpgrades > m_Upgrades;
	CUtlMap< const char*, int > m_AttribMap;
} */
#endif